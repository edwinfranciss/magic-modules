// Copyright 2024 Google Inc.
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package api

import (
	"fmt"
	"log"
	"strings"

	"github.com/GoogleCloudPlatform/magic-modules/mmv1/api/product"
	"github.com/GoogleCloudPlatform/magic-modules/mmv1/api/resource"
	"github.com/GoogleCloudPlatform/magic-modules/mmv1/google"
	"golang.org/x/exp/slices"
)

// Represents a property type
type Type struct {
	Name string `yaml:"name,omitempty"`

	// original value of :name before the provider override happens
	// same as :name if not overridden in provider
	ApiName string `yaml:"api_name,omitempty"`

	// TODO rewrite: improve the parsing of properties based on type in resource yaml files.
	Type string

	DefaultValue interface{} `yaml:"default_value,omitempty"`

	// Expected to follow the format as follows:
	//
	//	description: |
	//		This is a description of a field.
	//		If it comprises multiple lines, it must continue to be indented.
	//
	Description string `yaml:"description,omitempty"`

	Exclude bool `yaml:"exclude,omitempty"`

	// Add a deprecation message for a field that's been deprecated in the API
	// use the YAML chomping folding indicator (>-) if this is a multiline
	// string, as providers expect a single-line one w/o a newline.
	DeprecationMessage string `yaml:"deprecation_message,omitempty"`

	// Add a removed message for fields no longer supported in the API. This should
	// be used for fields supported in one version but have been removed from
	// a different version.
	RemovedMessage string `yaml:"removed_message,omitempty"`

	// If set value will not be sent to server on sync.
	// For nested fields, this also needs to be set on each descendant (ie. self,
	// child, etc.).
	Output bool `yaml:"output,omitempty"`

	// If set to true, changes in the field's value require recreating the
	// resource.
	// For nested fields, this only applies at the current level. This means
	// it should be explicitly added to each field that needs the ForceNew
	// behavior.
	Immutable bool `yaml:"immutable,omitempty"`

	// Indicates that this field is client-side only (aka virtual.)
	ClientSide bool `yaml:"client_side,omitempty"`

	// url_param_only will not send the field in the resource body and will
	// not attempt to read the field from the API response.
	// NOTE - this doesn't work for nested fields
	UrlParamOnly bool `yaml:"url_param_only,omitempty"`

	// For nested fields, this only applies within the parent.
	// For example, an optional parent can contain a required child.
	Required bool `yaml:"required,omitempty"`

	// Additional query Parameters to append to GET calls.
	ReadQueryParams string `yaml:"read_query_params,omitempty"`

	UpdateVerb string `yaml:"update_verb,omitempty"`

	UpdateUrl string `yaml:"update_url,omitempty"`

	// Some updates only allow updating certain fields at once (generally each
	// top-level field can be updated one-at-a-time). If this is set, we group
	// fields to update by (verb, url, fingerprint, id) instead of just
	// (verb, url, fingerprint), to allow multiple fields to reuse the same
	// endpoints.
	UpdateId string `yaml:"update_id,omitempty"`

	// The fingerprint value required to update this field. Downstreams should
	// GET the resource and parse the fingerprint value while doing each update
	// call. This ensures we can supply the fingerprint to each distinct
	// request.
	FingerprintName string `yaml:"fingerprint_name,omitempty"`

	// If true, we will include the empty value in requests made including
	// this attribute (both creates and updates).  This rarely needs to be
	// set to true, and corresponds to both the "NullFields" and
	// "ForceSendFields" concepts in the autogenerated API clients.
	SendEmptyValue bool `yaml:"send_empty_value,omitempty"`

	// [Optional] If true, empty nested objects are sent to / read from the
	// API instead of flattened to null.
	// The difference between this and send_empty_value is that send_empty_value
	// applies when the key of an object is empty; this applies when the values
	// are all nil / default. eg: "expiration: null" vs "expiration: {}"
	// In the case of Terraform, this occurs when a block in config has optional
	// values, and none of them are used. Terraform returns a nil instead of an
	// empty map[string]interface{} like we'd expect.
	AllowEmptyObject bool `yaml:"allow_empty_object,omitempty"`

	MinVersion string `yaml:"min_version,omitempty"`

	ExactVersion string `yaml:"exact_version,omitempty"`

	// A list of properties that conflict with this property. Uses the "lineage"
	// field to identify the property eg: parent.meta.label.foo
	Conflicts []string `yaml:"conflicts,omitempty"`

	// A list of properties that at least one of must be set.
	AtLeastOneOf []string `yaml:"at_least_one_of,omitempty"`

	// A list of properties that exactly one of must be set.
	ExactlyOneOf []string `yaml:"exactly_one_of,omitempty"`

	// A list of properties that are required to be set together.
	RequiredWith []string `yaml:"required_with,omitempty"`

	// Can only be overridden - we should never set this ourselves.
	NewType string `yaml:"-"`

	Properties []*Type `yaml:"properties,omitempty"`

	EnumValues []string `yaml:"enum_values,omitempty"`

	ExcludeDocsValues bool `yaml:"exclude_docs_values,omitempty"`

	// ====================
	// Array Fields
	// ====================
	ItemType *Type  `yaml:"item_type,omitempty"`
	MinSize  string `yaml:"min_size,omitempty"`
	MaxSize  string `yaml:"max_size,omitempty"`
	// Adds a ValidateFunc to the item schema
	ItemValidation resource.Validation `yaml:"item_validation,omitempty"`

	ParentName string `yaml:"parent_name,omitempty"`

	// ====================
	// ResourceRef Fields
	// ====================
	Resource string `yaml:"resource,omitempty"`
	Imports  string `yaml:"imports,omitempty"`

	// ====================
	// Terraform Overrides
	// ====================

	// Adds a DiffSuppressFunc to the schema
	DiffSuppressFunc string `yaml:"diff_suppress_func,omitempty"`

	StateFunc string `yaml:"state_func,omitempty"` // Adds a StateFunc to the schema

	Sensitive bool `yaml:"sensitive,omitempty"` // Adds `Sensitive: true` to the schema

	// Does not set this value to the returned API value.  Useful for fields
	// like secrets where the returned API value is not helpful.
	IgnoreRead bool `yaml:"ignore_read,omitempty"`

	// Adds a ValidateFunc to the schema
	Validation resource.Validation `yaml:"validation,omitempty"`

	// Indicates that this is an Array that should have Set diff semantics.
	UnorderedList bool `yaml:"unordered_list,omitempty"`

	IsSet bool `yaml:"is_set,omitempty"` // Uses a Set instead of an Array

	// Optional function to determine the unique ID of an item in the set
	// If not specified, schema.HashString (when elements are string) or
	// schema.HashSchema are used.
	SetHashFunc string `yaml:"set_hash_func,omitempty"`

	// if true, then we get the default value from the Google API if no value
	// is set in the terraform configuration for this field.
	// It translates to setting the field to Computed & Optional in the schema.
	// For nested fields, this only applies at the current level. This means
	// it should be explicitly added to each field that needs the defaulting
	// behavior.
	DefaultFromApi bool `yaml:"default_from_api,omitempty"`

	// https://github.com/hashicorp/terraform/pull/20837
	// Apply a ConfigMode of SchemaConfigModeAttr to the field.
	// This should be avoided for new fields, and only used with old ones.
	SchemaConfigModeAttr bool `yaml:"schema_config_mode_attr,omitempty"`

	// Names of fields that should be included in the updateMask.
	UpdateMaskFields []string `yaml:"update_mask_fields,omitempty"`

	// For a TypeMap, the expander function to call on the key.
	// Defaults to expandString.
	KeyExpander string `yaml:"key_expander,omitempty"`

	// For a TypeMap, the DSF to apply to the key.
	KeyDiffSuppressFunc string `yaml:"key_diff_suppress_func,omitempty"`

	// ====================
	// Map Fields
	// ====================
	// The type definition of the contents of the map.
	ValueType *Type `yaml:"value_type,omitempty"`

	// While the API doesn't give keys an explicit name, we specify one
	// because in Terraform the key has to be a property of the object.
	//
	// The name of the key. Used in the Terraform schema as a field name.
	KeyName string `yaml:"key_name,omitempty"`

	// A description of the key's format. Used in Terraform to describe
	// the field in documentation.
	KeyDescription string `yaml:"key_description,omitempty"`

	// ====================
	// KeyValuePairs Fields
	// ====================
	// Ignore writing the "effective_labels" and "effective_annotations" fields to API.
	IgnoreWrite bool `yaml:"ignore_write,omitempty"`

	// ====================
	// Schema Modifications
	// ====================
	// Schema modifications change the schema of a resource in some
	// fundamental way. They're not very portable, and will be hard to
	// generate so we should limit their use. Generally, if you're not
	// converting existing Terraform resources, these shouldn't be used.
	//
	// With great power comes great responsibility.

	// Flattens a NestedObject by removing that field from the Terraform
	// schema but will preserve it in the JSON sent/retrieved from the API
	//
	// EX: a API schema where fields are nested (eg: `one.two.three`) and we
	// desire the properties of the deepest nested object (eg: `three`) to
	// become top level properties in the Terraform schema. By overriding
	// the properties `one` and `one.two` and setting flatten_object then
	// all the properties in `three` will be at the root of the TF schema.
	//
	// We need this for cases where a field inside a nested object has a
	// default, if we can't spend a breaking change to fix a misshapen
	// field, or if the UX is _much_ better otherwise.
	//
	// WARN: only fully flattened properties are currently supported. In the
	// example above you could not flatten `one.two` without also flattening
	// all of it's parents such as `one`
	FlattenObject bool `yaml:"flatten_object,omitempty"`

	// ===========
	// Custom code
	// ===========
	// All custom code attributes are string-typed.  The string should
	// be the name of a template file which will be compiled in the
	// specified / described place.

	// A custom expander replaces the default expander for an attribute.
	// It is called as part of Create, and as part of Update if
	// object.input is false.  It can return an object of any type,
	// so the function header *is* part of the custom code template.
	// As with flatten, `property` and `prefix` are available.
	CustomExpand string `yaml:"custom_expand,omitempty"`

	// A custom flattener replaces the default flattener for an attribute.
	// It is called as part of Read.  It can return an object of any
	// type, and may sometimes need to return an object with non-interface{}
	// type so that the d.Set() call will succeed, so the function
	// header *is* a part of the custom code template.  To help with
	// creating the function header, `property` and `prefix` are available,
	// just as they are in the standard flattener template.
	CustomFlatten string `yaml:"custom_flatten,omitempty"`

	ResourceMetadata *Resource `yaml:"resource_metadata,omitempty"`

	ParentMetadata *Type `yaml:"parent_metadata,omitempty"` // is nil for top-level properties

	// The prefix used as part of the property expand/flatten function name
	// flatten{{$.GetPrefix}}{{$.TitlelizeProperty}}
	Prefix string `yaml:"prefix,omitempty"`
}

const MAX_NAME = 20

func (t *Type) SetDefault(r *Resource) {
	t.ResourceMetadata = r
	if t.UpdateVerb == "" {
		t.UpdateVerb = t.ResourceMetadata.UpdateVerb
	}

	switch {
	case t.IsA("Array"):
		t.ItemType.Name = t.Name
		t.ItemType.ParentName = t.Name
		t.ItemType.ParentMetadata = t
		t.ItemType.SetDefault(r)
	case t.IsA("Map"):
		if t.KeyExpander == "" {
			t.KeyExpander = "tpgresource.ExpandString"
		}
		t.ValueType.ParentName = t.Name
		t.ValueType.ParentMetadata = t
		t.ValueType.SetDefault(r)
	case t.IsA("NestedObject"):
		if t.Name == "" {
			t.Name = t.ParentName
		}

		if t.Description == "" {
			t.Description = "A nested object resource."
		}

		for _, p := range t.Properties {
			p.ParentMetadata = t
			p.SetDefault(r)
		}
	case t.IsA("ResourceRef"):
		if t.Name == "" {
			t.Name = t.Resource
		}

		if t.Description == "" {
			t.Description = fmt.Sprintf("A reference to %s resource", t.Resource)
		}
	case t.IsA("Fingerprint"):
		// Represents a fingerprint.  A fingerprint is an output-only
		// field used for optimistic locking during updates.
		// They are fetched from the GCP response.
		t.Output = true
	default:
	}

	if t.ApiName == "" {
		t.ApiName = t.Name
	}
}

func (t *Type) Validate(rName string) {
	if t.Name == "" {
		log.Fatalf("Missing `name` for proprty with type %s in resource %s", t.Type, rName)
	}

	if t.Output && t.Required {
		log.Fatalf("Property %s cannot be output and required at the same time in resource %s.", t.Name, rName)
	}

	if t.DefaultFromApi && t.DefaultValue != nil {
		log.Fatalf("'default_value' and 'default_from_api' cannot be both set in resource %s", rName)
	}

	t.validateLabelsField()

	switch {
	case t.IsA("Array"):
		t.ItemType.Validate(rName)
	case t.IsA("Map"):
		t.ValueType.Validate(rName)
	case t.IsA("NestedObject"):
		for _, p := range t.Properties {
			p.Validate(rName)
		}
	default:
	}
}

// TODO rewrite: add validations
// check :description, required: true
// check :update_verb, allowed: %i[POST PUT PATCH NONE],
// check_default_value_property
// check_conflicts
// check_at_least_one_of
// check_exactly_one_of
// check_required_with
// check the allowed types for Type field
// check the allowed fields for each type, for example, KeyName is only allowed for Map

// Prints a dot notation path to where the field is nested within the parent
// object. eg: parent.meta.label.foo
// The only intended purpose is to allow better error messages. Some objects
// and at some points in the build this doesn't output a valid output.
func (t Type) Lineage() string {
	if t.ParentMetadata == nil {
		return google.Underscore(t.Name)
	}

	return fmt.Sprintf("%s.%s", t.ParentMetadata.Lineage(), google.Underscore(t.Name))
}

// Returns the lineage in snake case
func (t Type) LineageAsSnakeCase() string {
	if t.ParentMetadata == nil {
		return google.Underscore(t.Name)
	}

	return fmt.Sprintf("%s_%s", t.ParentMetadata.LineageAsSnakeCase(), google.Underscore(t.Name))
}

// Prints the access path of the field in the configration eg: metadata.0.labels
// The only intended purpose is to get the value of the labes field by calling d.Get().
func (t Type) TerraformLineage() string {
	if t.ParentMetadata == nil || t.ParentMetadata.FlattenObject {
		return google.Underscore(t.Name)
	}

	return fmt.Sprintf("%s.0.%s", t.ParentMetadata.TerraformLineage(), google.Underscore(t.Name))
}

func (t Type) EnumValuesToString(quoteSeperator string, addEmpty bool) string {
	var values []string

	for _, val := range t.EnumValues {
		values = append(values, fmt.Sprintf("%s%s%s", quoteSeperator, val, quoteSeperator))
	}

	if addEmpty && !slices.Contains(values, "\"\"") && !t.Required {
		values = append(values, "\"\"")
	}

	return strings.Join(values, ", ")
}

func (t Type) TitlelizeProperty() string {
	return google.Camelize(t.Name, "upper")
}

// If the Prefix field is already set, returns the value.
// Otherwise, set the Prefix field and returns the value.
func (t *Type) GetPrefix() string {
	if t.Prefix == "" {
		if t.ParentMetadata == nil {
			nestedPrefix := ""
			// TODO: Use the nestedPrefix for tgc provider to be consistent with terraform provider
			if t.ResourceMetadata.NestedQuery != nil && t.ResourceMetadata.Compiler != "terraformgoogleconversion-codegen" {
				nestedPrefix = "Nested"
			}

			t.Prefix = fmt.Sprintf("%s%s", nestedPrefix, t.ResourceMetadata.ResourceName())
		} else {
			if t.ParentMetadata != nil && (t.ParentMetadata.IsA("Array") || t.ParentMetadata.IsA("Map")) {
				t.Prefix = t.ParentMetadata.GetPrefix()
			} else {
				if t.ParentMetadata != nil && t.ParentMetadata.ParentMetadata != nil && t.ParentMetadata.ParentMetadata.IsA("Map") {
					t.Prefix = fmt.Sprintf("%s%s", t.ParentMetadata.GetPrefix(), t.ParentMetadata.ParentMetadata.TitlelizeProperty())
				} else {
					t.Prefix = fmt.Sprintf("%s%s", t.ParentMetadata.GetPrefix(), t.ParentMetadata.TitlelizeProperty())
				}
			}
		}
	}
	return t.Prefix
}

func (t Type) ResourceType() string {
	r := t.ResourceRef()
	if r == nil {
		return ""
	}
	path := strings.Split(r.BaseUrl, "/")
	return path[len(path)-1]
}

// TODO rewrite: validation
// func (t *Type) check_default_value_property() {
// return if @default_value.nil?

// case self
// when Api::Type::String
//   clazz = ::String
// when Api::Type::Integer
//   clazz = ::Integer
// when Api::Type::Double
//   clazz = ::Float
// when Api::Type::Enum
//   clazz = ::Symbol
// when Api::Type::Boolean
//   clazz = :boolean
// when Api::Type::ResourceRef
//   clazz = [::String, ::Hash]
// else
//   raise "Update 'check_default_value_property' method to support " \
//         "default value for type //{self.class}"
// end

// check :default_value, type: clazz
// }

// Checks that all conflicting properties actually exist.
// This currently just returns if empty, because we don't want to do the check, since
// this list will have a full path for nested attributes.
// func (t *Type) check_conflicts() {
// check :conflicts, type: ::Array, default: [], item_type: ::String

// return if @conflicts.empty?
// }

// Returns list of properties that are in conflict with this property.
// func (t *Type) conflicting() {
func (t Type) Conflicting() []string {
	if t.ResourceMetadata == nil {
		return []string{}
	}

	return t.Conflicts
}

// TODO rewrite: validation
// Checks that all properties that needs at least one of their fields actually exist.
// This currently just returns if empty, because we don't want to do the check, since
// this list will have a full path for nested attributes.
// func (t *Type) check_at_least_one_of() {
// check :at_least_one_of, type: ::Array, default: [], item_type: ::String

// return if @at_least_one_of.empty?
// }

// Returns list of properties that needs at least one of their fields set.
// func (t *Type) at_least_one_of_list() {
func (t Type) AtLeastOneOfList() []string {
	if t.ResourceMetadata == nil {
		return []string{}
	}

	return t.AtLeastOneOf
}

// TODO rewrite: validation
// Checks that all properties that needs exactly one of their fields actually exist.
// This currently just returns if empty, because we don't want to do the check, since
// this list will have a full path for nested attributes.
// func (t *Type) check_exactly_one_of() {
// check :exactly_one_of, type: ::Array, default: [], item_type: ::String

// return if @exactly_one_of.empty?
// }

// Returns list of properties that needs exactly one of their fields set.
// func (t *Type) exactly_one_of_list() {
func (t Type) ExactlyOneOfList() []string {
	if t.ResourceMetadata == nil {
		return []string{}
	}

	return t.ExactlyOneOf
}

// TODO rewrite: validation
// Checks that all properties that needs required with their fields actually exist.
// This currently just returns if empty, because we don't want to do the check, since
// this list will have a full path for nested attributes.
// func (t *Type) check_required_with() {
// check :required_with, type: ::Array, default: [], item_type: ::String

// return if @required_with.empty?
// }

// Returns list of properties that needs required with their fields set.
func (t Type) RequiredWithList() []string {
	if t.ResourceMetadata == nil {
		return []string{}
	}

	return t.RequiredWith
}

func (t Type) Parent() *Type {
	return t.ParentMetadata
}

func (t Type) MinVersionObj() *product.Version {
	if t.MinVersion != "" {
		return t.ResourceMetadata.ProductMetadata.versionObj(t.MinVersion)
	} else {
		return t.ResourceMetadata.MinVersionObj()
	}
}

func (t *Type) exactVersionObj() *product.Version {
	if t.ExactVersion == "" {
		return nil
	}

	return t.ResourceMetadata.ProductMetadata.versionObj(t.ExactVersion)
}

func (t *Type) ExcludeIfNotInVersion(version *product.Version) {
	if !t.Exclude {
		if versionObj := t.exactVersionObj(); versionObj != nil {
			t.Exclude = versionObj.CompareTo(version) != 0
		}

		if !t.Exclude {
			t.Exclude = version.CompareTo(t.MinVersionObj()) < 0
		}
	}

	if t.IsA("NestedObject") {
		for _, p := range t.Properties {
			p.ExcludeIfNotInVersion(version)
		}
	} else if t.IsA("Array") && t.ItemType.IsA("NestedObject") {
		t.ItemType.ExcludeIfNotInVersion(version)
	}
}

func (t Type) IsA(clazz string) bool {
	if clazz == "" {
		log.Fatalf("class cannot be empty")
	}

	if t.NewType != "" {
		return t.NewType == clazz
	}

	return t.Type == clazz
}

// Returns nested properties for this property.
func (t Type) NestedProperties() []*Type {
	props := make([]*Type, 0)

	switch {
	case t.IsA("Array"):
		if t.ItemType.IsA("NestedObject") {
			props = google.Reject(t.ItemType.NestedProperties(), func(p *Type) bool {
				return t.Exclude
			})
		}
	case t.IsA("NestedObject"):
		props = t.UserProperties()
	case t.IsA("Map"):
		props = google.Reject(t.ValueType.NestedProperties(), func(p *Type) bool {
			return t.Exclude
		})
	default:
	}
	return props
}

func (t Type) Removed() bool {
	return t.RemovedMessage != ""
}

func (t Type) Deprecated() bool {
	return t.DeprecationMessage != ""
}

func (t *Type) GetDescription() string {
	return strings.TrimSpace(strings.TrimRight(t.Description, "\n"))
}

// TODO rewrite: validation
// class Array < Composite
//     check :item_type, type: [::String, NestedObject, ResourceRef, Enum], required: true

//     unless @item_type.is_a?(NestedObject) || @item_type.is_a?(ResourceRef) \
//         || @item_type.is_a?(Enum) || type?(@item_type)
//       raise "Invalid type //{@item_type}"
//     end

// This function is for array field
func (t Type) ItemTypeClass() string {
	if !t.IsA("Array") {
		return ""
	}

	return t.ItemType.Type
}

func (t Type) TFType(s string) string {
	switch s {
	case "Boolean":
		return "schema.TypeBool"
	case "Double":
		return "schema.TypeFloat"
	case "Integer":
		return "schema.TypeInt"
	case "String":
		return "schema.TypeString"
	case "Time":
		return "schema.TypeString"
	case "Enum":
		return "schema.TypeString"
	case "ResourceRef":
		return "schema.TypeString"
	case "NestedObject":
		return "schema.TypeList"
	case "Array":
		return "schema.TypeList"
	case "KeyValuePairs":
		return "schema.TypeMap"
	case "KeyValueLabels":
		return "schema.TypeMap"
	case "KeyValueTerraformLabels":
		return "schema.TypeMap"
	case "KeyValueEffectiveLabels":
		return "schema.TypeMap"
	case "KeyValueAnnotations":
		return "schema.TypeMap"
	case "Map":
		return "schema.TypeSet"
	case "Fingerprint":
		return "schema.TypeString"
	}

	return "schema.TypeString"
}

// TODO rewrite: validation
// // Represents an enum, and store is valid values
// class Enum < Primitive
//   values
//   skip_docs_values

//   func (t *Type) validate
//     super
//     check :values, type: ::Array, item_type: [Symbol, ::String, ::Integer], required: true
//     check :skip_docs_values, type: :boolean
//   end

// // Represents a reference to another resource
// class ResourceRef < Type
//   // The fields which can be overridden in provider.yaml.
//   module Fields
//     resource
//     imports
//   end
//   include Fields

//   func (t *Type) validate
//     super
//     @name = @resource if @name.nil?
//     @description = "A reference to //{@resource} resource" \
//       if @description.nil?

//     return if @__resource.nil? || @__resource.exclude || @exclude

//     check :resource, type: ::String, required: true
//     check :imports, type: ::String, required: TrueClass

//     // TODO: (camthornton) product reference may not exist yet
//     return if @__resource.__product.nil?

//     check_resource_ref_property_exists
//   end

func (t Type) ResourceRef() *Resource {
	if !t.IsA("ResourceRef") {
		return nil
	}

	product := t.ResourceMetadata.ProductMetadata
	resources := google.Select(product.Objects, func(obj *Resource) bool {
		return obj.Name == t.Resource
	})

	return resources[0]
}

// TODO rewrite: validation
//   func (t *Type) check_resource_ref_property_exists
//     return unless defined?(resource_ref.all_user_properties)

//     exported_props = resource_ref.all_user_properties
//     exported_props << Api::Type::String.new('selfLink') \
//       if resource_ref.has_self_link
//     raise "'//{@imports}' does not exist on '//{@resource}'" \
//       if exported_props.none? { |p| p.name == @imports }
//   end
// end

// // An structured object composed of other objects.
// class NestedObject < Composite

//   func (t *Type) validate
//     @description = 'A nested object resource' if @description.nil?
//     @name = @__name if @name.nil?
//     super

//     raise "Properties missing on //{name}" if @properties.nil?

//     @properties.each do |p|
//       p.set_variable(@__resource, :__resource)
//       p.set_variable(self, :__parent)
//     end
//     check :properties, type: ::Array, item_type: Api::Type, required: true
//   end

// Returns all properties including the ones that are excluded
// This is used for PropertyOverride validation
func (t Type) AllProperties() []*Type {
	return t.Properties
}

func (t Type) UserProperties() []*Type {
	if t.IsA("NestedObject") {
		if t.Properties == nil {
			log.Fatalf("Field '{%s}' properties are nil!", t.Lineage())
		}

		return google.Reject(t.Properties, func(p *Type) bool {
			return p.Exclude
		})
	}
	return nil
}

// Returns the list of top-level properties once any nested objects with
// flatten_object set to true have been collapsed
func (t *Type) RootProperties() []*Type {
	props := make([]*Type, 0)
	for _, p := range t.UserProperties() {
		if p.FlattenObject {
			props = google.Concat(props, p.RootProperties())
		} else {
			props = append(props, p)
		}
	}
	return props
}

// An array of string -> string key -> value pairs, such as labels.
// While this is technically a map, it's split out because it's a much
// simpler property to generate and means we can avoid conditional logic
// in Map.
func NewProperty(name, apiName string, options []func(*Type)) *Type {
	p := &Type{
		Name:    name,
		ApiName: apiName,
	}

	for _, option := range options {
		option(p)
	}
	return p
}

func propertyWithType(t string) func(*Type) {
	return func(p *Type) {
		p.Type = t
	}
}

func propertyWithOutput(output bool) func(*Type) {
	return func(p *Type) {
		p.Output = output
	}
}

func propertyWithDescription(description string) func(*Type) {
	return func(p *Type) {
		p.Description = description
	}
}

func propertyWithMinVersion(minVersion string) func(*Type) {
	return func(p *Type) {
		p.MinVersion = minVersion
	}
}

func propertyWithUpdateVerb(updateVerb string) func(*Type) {
	return func(p *Type) {
		p.UpdateVerb = updateVerb
	}
}

func propertyWithUpdateUrl(updateUrl string) func(*Type) {
	return func(p *Type) {
		p.UpdateUrl = updateUrl
	}
}

func propertyWithImmutable(immutable bool) func(*Type) {
	return func(p *Type) {
		p.Immutable = immutable
	}
}

func propertyWithClientSide(clientSide bool) func(*Type) {
	return func(p *Type) {
		p.ClientSide = clientSide
	}
}

func propertyWithIgnoreWrite(ignoreWrite bool) func(*Type) {
	return func(p *Type) {
		p.IgnoreWrite = ignoreWrite
	}
}

func (t *Type) validateLabelsField() {
	productName := t.ResourceMetadata.ProductMetadata.Name
	resourceName := t.ResourceMetadata.Name
	lineage := t.Lineage()
	if lineage == "labels" || lineage == "metadata.labels" || lineage == "configuration.labels" {
		if !t.IsA("KeyValueLabels") &&
			// The label value must be empty string, so skip this resource
			!(productName == "CloudIdentity" && resourceName == "Group") &&

			// The "labels" field has type Array, so skip this resource
			!(productName == "DeploymentManager" && resourceName == "Deployment") &&

			// https://github.com/hashicorp/terraform-provider-google/issues/16219
			!(productName == "Edgenetwork" && resourceName == "Network") &&

			// https://github.com/hashicorp/terraform-provider-google/issues/16219
			!(productName == "Edgenetwork" && resourceName == "Subnet") &&

			// "userLabels" is the resource labels field
			!(productName == "Monitoring" && resourceName == "NotificationChannel") &&

			// The "labels" field has type Array, so skip this resource
			!(productName == "Monitoring" && resourceName == "MetricDescriptor") {
			log.Fatalf("Please use type KeyValueLabels for field %s in resource %s/%s", lineage, productName, resourceName)
		}
	} else if t.IsA("KeyValueLabels") {
		log.Fatalf("Please don't use type KeyValueLabels for field %s in resource %s/%s", lineage, productName, resourceName)
	}

	if lineage == "annotations" || lineage == "metadata.annotations" {
		if !t.IsA("KeyValueAnnotations") &&
			// The "annotations" field has "ouput: true", so skip this eap resource
			!(productName == "Gkeonprem" && resourceName == "BareMetalAdminClusterEnrollment") {
			log.Fatalf("Please use type KeyValueAnnotations for field %s in resource %s/%s", lineage, productName, resourceName)
		}
	} else if t.IsA("KeyValueAnnotations") {
		log.Fatalf("Please don't use type KeyValueAnnotations for field %s in resource %s/%s", lineage, productName, resourceName)
	}
}

func (t Type) fieldMinVersion() string {
	return t.MinVersion
}

// TODO rewrite: validation
// // An array of string -> string key -> value pairs used specifically for the "labels" field.
// // The field name with this type should be "labels" literally.
// class KeyValueLabels < KeyValuePairs
//   func (t *Type) validate
//     super
//     return unless @name != 'labels'

//     raise "The field //{name} has the type KeyValueLabels, but the field name is not 'labels'!"
//   end
// end

// // An array of string -> string key -> value pairs used for the "terraform_labels" field.
// class KeyValueTerraformLabels < KeyValuePairs
// end

// // An array of string -> string key -> value pairs used for the "effective_labels"
// // and "effective_annotations" fields.
// class KeyValueEffectiveLabels < KeyValuePairs
// end

// // An array of string -> string key -> value pairs used specifically for the "annotations" field.
// // The field name with this type should be "annotations" literally.
// class KeyValueAnnotations < KeyValuePairs
//   func (t *Type) validate
//     super
//     return unless @name != 'annotations'

//     raise "The field //{name} has the type KeyValueAnnotations,\
// but the field name is not 'annotations'!"
//   end
// end

// TODO rewrite: validation
// // Map from string keys -> nested object entries
// class Map < Composite
//   func (t *Type) validate
//     super
//     check :key_name, type: ::String, required: true
//     check :key_description, type: ::String
//     check :value_type, type: Api::Type::NestedObject, required: true
//     raise "Invalid type //{@value_type}" unless type?(@value_type)
//   end

func (t Type) PropertyNsPrefix() []string {
	return []string{
		"Google",
		google.Camelize(t.ResourceMetadata.ProductMetadata.Name, "upper"),
		"Property",
	}
}

// "Namespace" - prefix with product and resource - a property with
// information from the "object" variable
func (t Type) NamespaceProperty() string {
	name := google.Camelize(t.Name, "upper")
	p := t
	for p.Parent() != nil {
		p = *p.Parent()
		name = fmt.Sprintf("%s%s", google.Camelize(p.Name, "upper"), name)
	}

	return fmt.Sprintf("%s%s%s", google.Camelize(t.ResourceMetadata.ProductMetadata.ApiName, "lower"), t.ResourceMetadata.Name, name)
}

func (t Type) CustomTemplate(templatePath string, appendNewline bool) string {
	return resource.ExecuteTemplate(&t, templatePath, appendNewline)
}

func (t *Type) GetIdFormat() string {
	return t.ResourceMetadata.GetIdFormat()
}

func (t *Type) GoLiteral(value interface{}) string {
	switch v := value.(type) {
	case int:
		return fmt.Sprintf("%d", v)
	case float64:
		return fmt.Sprintf("%.1f", v)
	case bool:
		return fmt.Sprintf("%v", v)
	case string:
		if !strings.HasPrefix(v, "\"") {
			return fmt.Sprintf("\"%s\"", v)
		}
		return v
	case []string:
		for i, val := range v {
			v[i] = fmt.Sprintf("\"%v\"", val)
		}
		return fmt.Sprintf("[]string{%s}", strings.Join(v, ","))

	default:
		panic(fmt.Errorf("unknown go literal type %+v", value))
	}
}

func (t *Type) IsForceNew() bool {
	if t.IsA("KeyValueLabels") && t.ResourceMetadata.RootLabels() {
		return false
	}

	if t.IsA("KeyValueTerraformLabels") && !t.ResourceMetadata.Updatable() && !t.ResourceMetadata.RootLabels() {
		return true
	}

	// Client-side fields don't inherit immutability
	if t.ClientSide {
		return t.Immutable
	}

	parent := t.Parent()
	return (!t.Output || t.IsA("KeyValueEffectiveLabels")) &&
		(t.Immutable ||
			(t.ResourceMetadata.Immutable && t.UpdateUrl == "" &&
				(parent == nil ||
					(parent.IsForceNew() &&
						!(parent.FlattenObject && t.IsA("KeyValueLabels"))))))
}

// Returns an updated path for a given Terraform field path (e.g.
// 'a_field', 'parent_field.0.child_name'). Returns nil if the property
// is not included in the resource's properties and removes keys that have
// been flattened
// FYI: Fields that have been renamed should use the new name, however, flattened
// fields still need to be included, ie:
// flattenedField > newParent > renameMe should be passed to this function as
// flattened_field.0.new_parent.0.im_renamed
// TODO(emilymye): Change format of input for
// exactly_one_of/at_least_one_of/etc to use camelcase, MM properities and
// convert to snake in this method
func (t *Type) GetPropertySchemaPath(schemaPath string) string {
	nestedProps := t.ResourceMetadata.UserProperites()

	var pathTkns []string
	for _, pname := range strings.Split(schemaPath, ".0.") {
		camelPname := google.Camelize(pname, "lower")
		index := slices.IndexFunc(nestedProps, func(p *Type) bool {
			return p.Name == camelPname
		})

		// if we couldn't find it, see if it was renamed at the top level
		if index == -1 {
			index = slices.IndexFunc(nestedProps, func(p *Type) bool {
				return p.Name == schemaPath
			})
		}

		if index == -1 {
			return ""
		}

		prop := nestedProps[index]

		nestedProps = prop.NestedProperties()
		if !prop.FlattenObject {
			pathTkns = append(pathTkns, google.Underscore(pname))
		}
	}

	if len(pathTkns) == 0 || pathTkns[len(pathTkns)-1] == "" {
		return ""
	}

	return strings.Join(pathTkns[:], ".0.")
}

func (t Type) GetPropertySchemaPathList(propertyList []string) []string {
	var list []string
	for _, path := range propertyList {
		path = t.GetPropertySchemaPath(path)
		if path != "" {
			list = append(list, path)
		}
	}
	return list
}
