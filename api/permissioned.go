package api

import (
	"github.com/filecoin-project/go-jsonrpc/auth"
	"reflect"
)

const (
	// When changing these, update docs/API.md too

	PermRead  auth.Permission = "read" // default
	PermWrite auth.Permission = "write"
	PermSign  auth.Permission = "sign"  // Use wallet keys for signing
	PermAdmin auth.Permission = "admin" // Manage permissions
)

var AllPermissions = []auth.Permission{PermRead, PermWrite, PermSign, PermAdmin}
var DefaultPerms = []auth.Permission{PermRead}

func permissionedProxies(in, out interface{}) {
	outs := GetInternalStructs(out)
	for _, o := range outs {
		auth.PermissionedProxy(AllPermissions, DefaultPerms, in, o)
	}
}

func PermissionedFullAPI(a GatewayFullNode) GatewayFullNode {
	var out GatewayFullNodeStruct
	permissionedProxies(a, &out)
	return &out
}

var _internalField = "Internal"

// GetInternalStructs extracts all pointers to 'Internal' sub-structs from the provided pointer to a proxy struct
func GetInternalStructs(in interface{}) []interface{} {
	return getInternalStructs(reflect.ValueOf(in).Elem())
}

func getInternalStructs(rv reflect.Value) []interface{} {
	var out []interface{}

	internal := rv.FieldByName(_internalField)
	ii := internal.Addr().Interface()
	out = append(out, ii)

	for i := 0; i < rv.NumField(); i++ {
		if rv.Type().Field(i).Name == _internalField {
			continue
		}

		sub := getInternalStructs(rv.Field(i))

		out = append(out, sub...)
	}

	return out
}
