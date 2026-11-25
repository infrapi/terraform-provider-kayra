package provider

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// kayraResource is one of Kayra's objects as a Terraform resource: d says
// which, and how its attributes map to its contract's schema.
type kayraResource struct {
	d      resourceDef
	client *client
}

var (
	_ resource.ResourceWithConfigure   = (*kayraResource)(nil)
	_ resource.ResourceWithImportState = (*kayraResource)(nil)
)

func (r *kayraResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_" + r.d.Type
}

func (r *kayraResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = resourceSchema(r.d)
}

func (r *kayraResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	c, ok := req.ProviderData.(*client)
	if !ok {
		resp.Diagnostics.AddError("Unexpected provider data", fmt.Sprintf("%T, not the Kayra client", req.ProviderData))
		return
	}
	r.client = c
}

// paths are v's attributes the paths name: its keys and path parameters.
func (r *kayraResource) paths(v tftypes.Value) (map[string]string, error) {
	attrs, err := objectAttrs(v)
	if err != nil {
		return nil, err
	}
	out := map[string]string{}
	for name, a := range attrs {
		if !a.Type().Is(tftypes.String) || !a.IsKnown() || a.IsNull() {
			continue
		}
		var s string
		if err := a.As(&s); err != nil {
			return nil, err
		}
		out[name] = s
	}
	return out, nil
}

// answer turns Kayra's object, m, into the state, prior the plan or the
// state it answers, its id from its keys.
func (r *kayraResource) answer(m map[string]any, prior tftypes.Value) (tftypes.Value, error) {
	fields := r.d.Fields
	v, err := state(fields, m, prior)
	if err != nil {
		return v, err
	}
	if hasField(fields, "id") {
		return v, nil
	}
	attrs, err := objectAttrs(v)
	if err != nil {
		return v, err
	}
	id, err := keysID(r.d.Keys, attrs)
	if err != nil {
		return v, err
	}
	attrs["id"] = tftypes.NewValue(tftypes.String, id)
	return tftypes.NewValue(objectType(r.d.allFields()), attrs), nil
}

func (r *kayraResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	in, err := body(r.d.Fields, req.Plan.Raw)
	if err == nil {
		var p string
		var attrs map[string]string
		if attrs, err = r.paths(req.Plan.Raw); err == nil {
			p, err = r.client.path(r.d.API, r.d.Collection, attrs)
		}
		if err == nil {
			var out map[string]any
			if err = r.client.do(ctx, http.MethodPost, p, in, &out); err == nil {
				resp.State.Raw, err = r.answer(out, req.Plan.Raw)
			}
		}
	}
	addError(&resp.Diagnostics, "Creating kayra_"+r.d.Type, err)
}

func (r *kayraResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	attrs, err := r.paths(req.State.Raw)
	if err == nil {
		var p string
		if p, err = r.client.path(r.d.API, r.d.Item, attrs); err == nil {
			var out map[string]any
			err = r.client.do(ctx, http.MethodGet, p, nil, &out)
			if errors.Is(err, errNotFound) {
				resp.State.RemoveResource(ctx)
				return
			}
			if err == nil {
				resp.State.Raw, err = r.answer(out, req.State.Raw)
			}
		}
	}
	addError(&resp.Diagnostics, "Reading kayra_"+r.d.Type, err)
}

func (r *kayraResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	if !r.d.Updatable {
		resp.Diagnostics.AddError("Updating kayra_"+r.d.Type, "Kayra replaces it: every change replaces it, the plan should have")
		return
	}
	in, err := body(r.d.Fields, req.Plan.Raw)
	if err == nil {
		var attrs map[string]string
		var p string
		if attrs, err = r.paths(req.State.Raw); err == nil {
			p, err = r.client.path(r.d.API, r.d.Item, attrs)
		}
		if err == nil {
			var out map[string]any
			if err = r.client.do(ctx, http.MethodPut, p, in, &out); err == nil {
				resp.State.Raw, err = r.answer(out, req.Plan.Raw)
			}
		}
	}
	addError(&resp.Diagnostics, "Updating kayra_"+r.d.Type, err)
}

func (r *kayraResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	attrs, err := r.paths(req.State.Raw)
	if err == nil {
		var p string
		if p, err = r.client.path(r.d.API, r.d.Item, attrs); err == nil {
			if err = r.client.do(ctx, http.MethodDelete, p, nil, nil); errors.Is(err, errNotFound) {
				err = nil
			}
		}
	}
	addError(&resp.Diagnostics, "Deleting kayra_"+r.d.Type, err)
}

// ImportState takes the id as its keys joined by /: acme/admins for a
// namespace's group, compute.infrapi.io for a registration.
func (r *kayraResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.Split(req.ID, "/")
	if len(parts) != len(r.d.Keys) || contains(parts, "") {
		resp.Diagnostics.AddError("Importing kayra_"+r.d.Type,
			fmt.Sprintf("id %q: %s", req.ID, strings.Join(r.d.Keys, "/")))
		return
	}
	for i, k := range r.d.Keys {
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root(k), parts[i])...)
	}
}

// addError reports err, Kayra's problem with its solution.
func addError(diags *diag.Diagnostics, summary string, err error) {
	if err == nil {
		return
	}
	var e *apiError
	if errors.As(err, &e) {
		detail := e.Problem.Detail
		if detail == "" {
			detail = e.Problem.Title
		}
		if e.Problem.Solution != "" {
			detail += "\n\nSolution: " + e.Problem.Solution
		}
		diags.AddError(fmt.Sprintf("%s: Kayra answered %d %s", summary, e.Problem.Status, e.Problem.Title),
			fmt.Sprintf("%s %s\n\n%s", e.Method, e.Path, detail))
		return
	}
	diags.AddError(summary, err.Error())
}
