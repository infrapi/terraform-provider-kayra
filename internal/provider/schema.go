package provider

import (
	dschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/mapplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// idField is the id every resource has but a token, whose own id it is:
// its keys joined by /, its import id.
var idField = field{Attr: "id", Kind: kString, Desc: "Its id, its import id: its keys joined by /.", ReadOnly: true, Stable: true}

// allFields are d's fields, its id first.
func (d resourceDef) allFields() []field {
	if hasField(d.Fields, "id") {
		return d.Fields
	}
	return append([]field{idField}, d.Fields...)
}

// modifiers is what a plan does with an attribute: replace the resource
// when it changes, or keep the state's value.
type modifiers struct{ replace, keep bool }

// resourceSchema is d's schema: an attribute per field, required as the
// contract requires it, computed when Kayra sets it; a key, or any input of
// a resource Kayra does not update, replaces it.
func resourceSchema(d resourceDef) schema.Schema {
	attrs := map[string]schema.Attribute{}
	for _, f := range d.allFields() {
		replace := !f.ReadOnly && (!d.Updatable || contains(d.Keys, f.Attr))
		attrs[f.Attr] = resourceAttr(f, modifiers{replace: replace, keep: f.Stable || f.Default})
	}
	return schema.Schema{Description: d.Desc, MarkdownDescription: d.Desc, Attributes: attrs}
}

func resourceAttr(f field, m modifiers) schema.Attribute {
	required := f.Required
	optional := !f.Required && !f.ReadOnly
	computed := f.ReadOnly || f.Default
	switch f.Kind {
	case kBool:
		var pm []planmodifier.Bool
		if m.replace {
			pm = append(pm, boolplanmodifier.RequiresReplace())
		}
		if m.keep {
			pm = append(pm, boolplanmodifier.UseStateForUnknown())
		}
		return schema.BoolAttribute{Description: f.Desc, MarkdownDescription: f.Desc, Required: required, Optional: optional,
			Computed: computed, Sensitive: f.Sensitive, PlanModifiers: pm}
	case kInt:
		var pm []planmodifier.Int64
		if m.replace {
			pm = append(pm, int64planmodifier.RequiresReplace())
		}
		if m.keep {
			pm = append(pm, int64planmodifier.UseStateForUnknown())
		}
		return schema.Int64Attribute{Description: f.Desc, MarkdownDescription: f.Desc, Required: required, Optional: optional,
			Computed: computed, Sensitive: f.Sensitive, PlanModifiers: pm}
	case kStrings:
		var pm []planmodifier.List
		if m.replace {
			pm = append(pm, listplanmodifier.RequiresReplace())
		}
		if m.keep {
			pm = append(pm, listplanmodifier.UseStateForUnknown())
		}
		return schema.ListAttribute{Description: f.Desc, MarkdownDescription: f.Desc, ElementType: types.StringType, Required: required,
			Optional: optional, Computed: computed, Sensitive: f.Sensitive, PlanModifiers: pm}
	case kStringMap:
		var pm []planmodifier.Map
		if m.replace {
			pm = append(pm, mapplanmodifier.RequiresReplace())
		}
		if m.keep {
			pm = append(pm, mapplanmodifier.UseStateForUnknown())
		}
		return schema.MapAttribute{Description: f.Desc, MarkdownDescription: f.Desc, ElementType: types.StringType, Required: required,
			Optional: optional, Computed: computed, Sensitive: f.Sensitive, PlanModifiers: pm}
	case kObject:
		var pm []planmodifier.Object
		if m.replace {
			pm = append(pm, objectplanmodifier.RequiresReplace())
		}
		if m.keep {
			pm = append(pm, objectplanmodifier.UseStateForUnknown())
		}
		return schema.SingleNestedAttribute{Description: f.Desc, MarkdownDescription: f.Desc, Attributes: nestedAttrs(f.Fields),
			Required: required, Optional: optional, Computed: computed, Sensitive: f.Sensitive, PlanModifiers: pm}
	case kObjects:
		var pm []planmodifier.List
		if m.replace {
			pm = append(pm, listplanmodifier.RequiresReplace())
		}
		if m.keep {
			pm = append(pm, listplanmodifier.UseStateForUnknown())
		}
		return schema.ListNestedAttribute{Description: f.Desc, MarkdownDescription: f.Desc,
			NestedObject: schema.NestedAttributeObject{Attributes: nestedAttrs(f.Fields)},
			Required:     required, Optional: optional, Computed: computed, Sensitive: f.Sensitive, PlanModifiers: pm}
	}
	var pm []planmodifier.String
	if m.replace {
		pm = append(pm, stringplanmodifier.RequiresReplace())
	}
	if m.keep {
		pm = append(pm, stringplanmodifier.UseStateForUnknown())
	}
	return schema.StringAttribute{Description: f.Desc, MarkdownDescription: f.Desc, Required: required, Optional: optional,
		Computed: computed, Sensitive: f.Sensitive, PlanModifiers: pm}
}

// nestedAttrs are an object's attributes: the resource's plan modifiers
// apply to the object as a whole.
func nestedAttrs(fields []field) map[string]schema.Attribute {
	attrs := map[string]schema.Attribute{}
	for _, f := range fields {
		attrs[f.Attr] = resourceAttr(f, modifiers{keep: f.Default})
	}
	return attrs
}

// dataSourceSchema is d's schema: its keys required, every other
// attribute computed.
func dataSourceSchema(d dataSourceDef) dschema.Schema {
	attrs := map[string]dschema.Attribute{}
	for _, f := range d.Fields {
		attrs[f.Attr] = dataSourceAttr(f, contains(d.Keys, f.Attr))
	}
	return dschema.Schema{Description: d.Desc, MarkdownDescription: d.Desc, Attributes: attrs}
}

func dataSourceAttr(f field, key bool) dschema.Attribute {
	computed := !key
	switch f.Kind {
	case kBool:
		return dschema.BoolAttribute{Description: f.Desc, MarkdownDescription: f.Desc, Required: key, Computed: computed, Sensitive: f.Sensitive}
	case kInt:
		return dschema.Int64Attribute{Description: f.Desc, MarkdownDescription: f.Desc, Required: key, Computed: computed, Sensitive: f.Sensitive}
	case kStrings:
		return dschema.ListAttribute{Description: f.Desc, MarkdownDescription: f.Desc, ElementType: types.StringType, Computed: true}
	case kStringMap:
		return dschema.MapAttribute{Description: f.Desc, MarkdownDescription: f.Desc, ElementType: types.StringType, Computed: true}
	case kObject:
		return dschema.SingleNestedAttribute{Description: f.Desc, MarkdownDescription: f.Desc, Attributes: dataSourceNested(f.Fields), Computed: true}
	case kObjects:
		return dschema.ListNestedAttribute{Description: f.Desc, MarkdownDescription: f.Desc,
			NestedObject: dschema.NestedAttributeObject{Attributes: dataSourceNested(f.Fields)}, Computed: true}
	}
	return dschema.StringAttribute{Description: f.Desc, MarkdownDescription: f.Desc, Required: key, Computed: computed, Sensitive: f.Sensitive}
}

func dataSourceNested(fields []field) map[string]dschema.Attribute {
	attrs := map[string]dschema.Attribute{}
	for _, f := range fields {
		attrs[f.Attr] = dataSourceAttr(f, false)
	}
	return attrs
}

// tfType is f's value's type, as Terraform passes it.
func tfType(f field) tftypes.Type {
	switch f.Kind {
	case kBool:
		return tftypes.Bool
	case kInt:
		return tftypes.Number
	case kStrings:
		return tftypes.List{ElementType: tftypes.String}
	case kStringMap:
		return tftypes.Map{ElementType: tftypes.String}
	case kObject:
		return objectType(f.Fields)
	case kObjects:
		return tftypes.List{ElementType: objectType(f.Fields)}
	}
	return tftypes.String
}

func objectType(fields []field) tftypes.Object {
	t := tftypes.Object{AttributeTypes: map[string]tftypes.Type{}}
	for _, f := range fields {
		t.AttributeTypes[f.Attr] = tfType(f)
	}
	return t
}

func contains(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}
