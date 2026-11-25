package provider

// kind is how a field is typed, as its contract types it.
type kind int

const (
	kString    kind = iota // a string, a date-time too
	kBool                  // a boolean
	kInt                   // an integer
	kStrings               // an array of strings
	kStringMap             // an object of strings
	kObject                // an object: its Fields
	kObjects               // an array of objects: its Fields
)

// field is one property of a contract's schema, an attribute of the
// resource or data source, under the property's name.
type field struct {
	Attr string
	// JSON is its property's name, when it is not Attr: a Terraform
	// meta-argument's, depends_on.
	JSON string
	Desc string
	Kind kind
	// Required: the contract requires it, and it is not read-only.
	Required bool
	// ReadOnly: Kayra sets it, a request never sends it: computed.
	ReadOnly bool
	// WriteOnly: a request sends it, Kayra never returns it: the state
	// keeps the configuration's.
	WriteOnly bool
	// Default: Kayra sets it when the request leaves it out: optional and
	// computed.
	Default bool
	// PathOnly: a parameter of the path, not of the schema: the namespace.
	PathOnly bool
	// Stable: read-only, and never changes once set: the plan keeps it.
	Stable bool
	// Sensitive: the state must not show it.
	Sensitive bool
	Fields    []field
}

// resourceDef is a Terraform resource, kayra_<Type>, over one of Kayra's
// schemas: created by a POST on Collection, read, replaced and deleted on
// Item, both relative to API's root (/.well-known/kayra.json), {attr} an
// attribute's value.
type resourceDef struct {
	Type, API, Collection, Item string
	// Keys are the attributes naming it, its import id's parts joined by /.
	Keys []string
	// Updatable: the contract replaces it (PUT); otherwise every change
	// replaces it.
	Updatable bool
	Desc      string
	Fields    []field
}

// dataSourceDef is a Terraform data source, kayra_<Type>: a GET of Path.
type dataSourceDef struct {
	Type, API, Path string
	Keys            []string
	Desc            string
	Fields          []field
}

// name is f's property's name.
func (f field) name() string {
	if f.JSON != "" {
		return f.JSON
	}
	return f.Attr
}

// hasField tells whether fields have one of attr.
func hasField(fields []field, attr string) bool {
	for _, f := range fields {
		if f.Attr == attr {
			return true
		}
	}
	return false
}
