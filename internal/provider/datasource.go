package provider

import (
	"context"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
)

// kayraDataSource reads one of Kayra's objects: d says which.
type kayraDataSource struct {
	d      dataSourceDef
	client *client
}

var _ datasource.DataSourceWithConfigure = (*kayraDataSource)(nil)

func (s *kayraDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_" + s.d.Type
}

func (s *kayraDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = dataSourceSchema(s.d)
}

func (s *kayraDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	c, ok := req.ProviderData.(*client)
	if !ok {
		resp.Diagnostics.AddError("Unexpected provider data", fmt.Sprintf("%T, not the Kayra client", req.ProviderData))
		return
	}
	s.client = c
}

func (s *kayraDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	r := &kayraResource{d: resourceDef{Keys: s.d.Keys}}
	attrs, err := r.paths(req.Config.Raw)
	if err == nil {
		var p string
		if p, err = s.client.path(s.d.API, s.d.Path, attrs); err == nil {
			var out map[string]any
			if err = s.client.do(ctx, http.MethodGet, p, nil, &out); err == nil {
				resp.State.Raw, err = state(s.d.Fields, out, req.Config.Raw)
			}
		}
	}
	addError(&resp.Diagnostics, "Reading data kayra_"+s.d.Type, err)
}
