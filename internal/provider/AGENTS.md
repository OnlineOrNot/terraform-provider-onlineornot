# Provider Package

Resources, data sources, and generated schema code for terraform-provider-onlineornot.

## Structure

```
provider/
├── provider.go                    # Provider definition, Configure, Resources(), DataSources()
├── *_resource.go                  # Manual CRUD implementations (18 files)
├── *_data_source.go               # Data source implementations (7 files)
└── resource_*/                    # Generated schema packages (DO NOT EDIT)
    └── *_resource_gen.go
```

## Where to Look

| Task                              | File                                                     |
| --------------------------------- | -------------------------------------------------------- |
| Register new resource/data source | `provider.go` → `Resources()` / `DataSources()`          |
| Resource CRUD logic               | `<name>_resource.go` (e.g., `check_resource.go`)         |
| Data source read logic            | `<name>_data_source.go` or `<name>s_data_source.go`      |
| Schema/model definitions          | `resource_<name>/<name>_resource_gen.go` (generated)     |
| Nested resource imports           | Look for `ImportState` with `strings.Split(req.ID, "/")` |

## Patterns

### Resource Implementation

```go
var _ resource.Resource = &CheckResource{}
var _ resource.ResourceWithImportState = &CheckResource{}

type CheckResource struct { client *client.Client }

func NewCheckResource() resource.Resource { return &CheckResource{} }

// Methods: Metadata, Schema, Configure, Create, Read, Update, Delete, ImportState
```

### Schema from Generated Package

```go
func (r *CheckResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
    resp.Schema = resource_check.CheckResourceSchema(ctx)
}
```

### Model Conversion (Terraform ↔ API)

```go
// Terraform → API
check := &client.Check{
    Name: data.Name.ValueString(),
    URL:  data.Url.ValueString(),
}
if !data.FollowRedirects.IsNull() {
    v := data.FollowRedirects.ValueBool()
    check.FollowRedirects = &v
}

// API → Terraform
data.Id = types.StringValue(check.ID)
data.Name = types.StringValue(check.Name)
```

### Nested Resource Import (status_page_id/component_id)

```go
parts := strings.Split(req.ID, "/")
resp.State.SetAttribute(ctx, path.Root("status_page_id"), parts[0])
resp.State.SetAttribute(ctx, path.Root("id"), parts[1])
```

## Anti-Patterns

### Generated Code

Files in `resource_*/` are generated - edit `generator_config.yml` instead.

### IsUnknown() Check

Always check before setting null:

```go
// CORRECT
if data.Field.IsUnknown() {
    data.Field = types.StringNull()
}

// WRONG - overwrites user values
data.Field = types.StringNull()
```

### Complex List Null Types

For nested object lists, construct the element type:

```go
if data.Components.IsUnknown() {
    elemType := resource_status_page_incident.ComponentsType{
        ObjectType: types.ObjectType{
            AttrTypes: resource_status_page_incident.ComponentsValue{}.AttributeTypes(ctx),
        },
    }
    data.Components = types.ListNull(elemType)
}
```
