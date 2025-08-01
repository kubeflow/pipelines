// Code generated by go-swagger; DO NOT EDIT.

package experiment_model

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"context"
	"encoding/json"

	"github.com/go-openapi/strfmt"
	"github.com/go-openapi/swag"
)

// ProtobufAny `Any` contains an arbitrary serialized protocol buffer message along with a
// URL that describes the type of the serialized message.
//
// Protobuf library provides support to pack/unpack Any values in the form
// of utility functions or additional generated methods of the Any type.
//
// Example 1: Pack and unpack a message in C++.
//
//	Foo foo = ...;
//	Any any;
//	any.PackFrom(foo);
//	...
//	if (any.UnpackTo(&foo)) {
//	  ...
//	}
//
// Example 2: Pack and unpack a message in Java.
//
//	   Foo foo = ...;
//	   Any any = Any.pack(foo);
//	   ...
//	   if (any.is(Foo.class)) {
//	     foo = any.unpack(Foo.class);
//	   }
//	   // or ...
//	   if (any.isSameTypeAs(Foo.getDefaultInstance())) {
//	     foo = any.unpack(Foo.getDefaultInstance());
//	   }
//
//	Example 3: Pack and unpack a message in Python.
//
//	   foo = Foo(...)
//	   any = Any()
//	   any.Pack(foo)
//	   ...
//	   if any.Is(Foo.DESCRIPTOR):
//	     any.Unpack(foo)
//	     ...
//
//	Example 4: Pack and unpack a message in Go
//
//	    foo := &pb.Foo{...}
//	    any, err := anypb.New(foo)
//	    if err != nil {
//	      ...
//	    }
//	    ...
//	    foo := &pb.Foo{}
//	    if err := any.UnmarshalTo(foo); err != nil {
//	      ...
//	    }
//
// The pack methods provided by protobuf library will by default use
// 'type.googleapis.com/full.type.name' as the type URL and the unpack
// methods only use the fully qualified type name after the last '/'
// in the type URL, for example "foo.bar.com/x/y.z" will yield type
// name "y.z".
//
// JSON
// ====
// The JSON representation of an `Any` value uses the regular
// representation of the deserialized, embedded message, with an
// additional field `@type` which contains the type URL. Example:
//
//	package google.profile;
//	message Person {
//	  string first_name = 1;
//	  string last_name = 2;
//	}
//
//	{
//	  "@type": "type.googleapis.com/google.profile.Person",
//	  "firstName": <string>,
//	  "lastName": <string>
//	}
//
// If the embedded message type is well-known and has a custom JSON
// representation, that representation will be embedded adding a field
// `value` which holds the custom JSON in addition to the `@type`
// field. Example (for message [google.protobuf.Duration][]):
//
//	{
//	  "@type": "type.googleapis.com/google.protobuf.Duration",
//	  "value": "1.212s"
//	}
//
// swagger:model protobufAny
type ProtobufAny struct {

	// A URL/resource name that uniquely identifies the type of the serialized
	// protocol buffer message. This string must contain at least
	// one "/" character. The last segment of the URL's path must represent
	// the fully qualified name of the type (as in
	// `path/google.protobuf.Duration`). The name should be in a canonical form
	// (e.g., leading "." is not accepted).
	//
	// In practice, teams usually precompile into the binary all types that they
	// expect it to use in the context of Any. However, for URLs which use the
	// scheme `http`, `https`, or no scheme, one can optionally set up a type
	// server that maps type URLs to message definitions as follows:
	//
	// * If no scheme is provided, `https` is assumed.
	// * An HTTP GET on the URL must yield a [google.protobuf.Type][]
	//   value in binary format, or produce an error.
	// * Applications are allowed to cache lookup results based on the
	//   URL, or have them precompiled into a binary to avoid any
	//   lookup. Therefore, binary compatibility needs to be preserved
	//   on changes to types. (Use versioned type names to manage
	//   breaking changes.)
	//
	// Note: this functionality is not currently available in the official
	// protobuf release, and it is not used for type URLs beginning with
	// type.googleapis.com. As of May 2023, there are no widely used type server
	// implementations and no plans to implement one.
	//
	// Schemes other than `http`, `https` (or the empty scheme) might be
	// used with implementation specific semantics.
	AtType string `json:"@type,omitempty"`

	// protobuf any
	ProtobufAny map[string]interface{} `json:"-"`
}

// UnmarshalJSON unmarshals this object with additional properties from JSON
func (m *ProtobufAny) UnmarshalJSON(data []byte) error {
	// stage 1, bind the properties
	var stage1 struct {

		// A URL/resource name that uniquely identifies the type of the serialized
		// protocol buffer message. This string must contain at least
		// one "/" character. The last segment of the URL's path must represent
		// the fully qualified name of the type (as in
		// `path/google.protobuf.Duration`). The name should be in a canonical form
		// (e.g., leading "." is not accepted).
		//
		// In practice, teams usually precompile into the binary all types that they
		// expect it to use in the context of Any. However, for URLs which use the
		// scheme `http`, `https`, or no scheme, one can optionally set up a type
		// server that maps type URLs to message definitions as follows:
		//
		// * If no scheme is provided, `https` is assumed.
		// * An HTTP GET on the URL must yield a [google.protobuf.Type][]
		//   value in binary format, or produce an error.
		// * Applications are allowed to cache lookup results based on the
		//   URL, or have them precompiled into a binary to avoid any
		//   lookup. Therefore, binary compatibility needs to be preserved
		//   on changes to types. (Use versioned type names to manage
		//   breaking changes.)
		//
		// Note: this functionality is not currently available in the official
		// protobuf release, and it is not used for type URLs beginning with
		// type.googleapis.com. As of May 2023, there are no widely used type server
		// implementations and no plans to implement one.
		//
		// Schemes other than `http`, `https` (or the empty scheme) might be
		// used with implementation specific semantics.
		AtType string `json:"@type,omitempty"`
	}
	if err := json.Unmarshal(data, &stage1); err != nil {
		return err
	}
	var rcv ProtobufAny

	rcv.AtType = stage1.AtType
	*m = rcv

	// stage 2, remove properties and add to map
	stage2 := make(map[string]json.RawMessage)
	if err := json.Unmarshal(data, &stage2); err != nil {
		return err
	}

	delete(stage2, "@type")
	// stage 3, add additional properties values
	if len(stage2) > 0 {
		result := make(map[string]interface{})
		for k, v := range stage2 {
			var toadd interface{}
			if err := json.Unmarshal(v, &toadd); err != nil {
				return err
			}
			result[k] = toadd
		}
		m.ProtobufAny = result
	}

	return nil
}

// MarshalJSON marshals this object with additional properties into a JSON object
func (m ProtobufAny) MarshalJSON() ([]byte, error) {
	var stage1 struct {

		// A URL/resource name that uniquely identifies the type of the serialized
		// protocol buffer message. This string must contain at least
		// one "/" character. The last segment of the URL's path must represent
		// the fully qualified name of the type (as in
		// `path/google.protobuf.Duration`). The name should be in a canonical form
		// (e.g., leading "." is not accepted).
		//
		// In practice, teams usually precompile into the binary all types that they
		// expect it to use in the context of Any. However, for URLs which use the
		// scheme `http`, `https`, or no scheme, one can optionally set up a type
		// server that maps type URLs to message definitions as follows:
		//
		// * If no scheme is provided, `https` is assumed.
		// * An HTTP GET on the URL must yield a [google.protobuf.Type][]
		//   value in binary format, or produce an error.
		// * Applications are allowed to cache lookup results based on the
		//   URL, or have them precompiled into a binary to avoid any
		//   lookup. Therefore, binary compatibility needs to be preserved
		//   on changes to types. (Use versioned type names to manage
		//   breaking changes.)
		//
		// Note: this functionality is not currently available in the official
		// protobuf release, and it is not used for type URLs beginning with
		// type.googleapis.com. As of May 2023, there are no widely used type server
		// implementations and no plans to implement one.
		//
		// Schemes other than `http`, `https` (or the empty scheme) might be
		// used with implementation specific semantics.
		AtType string `json:"@type,omitempty"`
	}

	stage1.AtType = m.AtType

	// make JSON object for known properties
	props, err := json.Marshal(stage1)
	if err != nil {
		return nil, err
	}

	if len(m.ProtobufAny) == 0 { // no additional properties
		return props, nil
	}

	// make JSON object for the additional properties
	additional, err := json.Marshal(m.ProtobufAny)
	if err != nil {
		return nil, err
	}

	if len(props) < 3 { // "{}": only additional properties
		return additional, nil
	}

	// concatenate the 2 objects
	return swag.ConcatJSON(props, additional), nil
}

// Validate validates this protobuf any
func (m *ProtobufAny) Validate(formats strfmt.Registry) error {
	return nil
}

// ContextValidate validates this protobuf any based on context it is used
func (m *ProtobufAny) ContextValidate(ctx context.Context, formats strfmt.Registry) error {
	return nil
}

// MarshalBinary interface implementation
func (m *ProtobufAny) MarshalBinary() ([]byte, error) {
	if m == nil {
		return nil, nil
	}
	return swag.WriteJSON(m)
}

// UnmarshalBinary interface implementation
func (m *ProtobufAny) UnmarshalBinary(b []byte) error {
	var res ProtobufAny
	if err := swag.ReadJSON(b, &res); err != nil {
		return err
	}
	*m = res
	return nil
}
