package api

import (
	"context"
	"reflect"

	"github.com/filecoin-project/go-jsonrpc/auth"
	"golang.org/x/xerrors"
)

type MethodName = string

var AllPermissions = []auth.Permission{"read", "write", "sign", "admin"}
var defaultPerms = []auth.Permission{"read"}

// permissionVerify the scheduler between API and internal business
func PermissionProxy(in interface{}, out interface{}) {
	ra := reflect.ValueOf(in)
	rint := reflect.ValueOf(out).Elem()
	for i := 0; i < ra.NumMethod(); i++ {
		methodName := ra.Type().Method(i).Name
		field, exists := rint.Type().FieldByName(methodName)
		if !exists {
			continue
		}

		requiredPerm := field.Tag.Get("perm")
		if requiredPerm == "" {
			panic("missing 'perm' tag on " + field.Name) // ok
		}

		fn := ra.Method(i)
		rint.FieldByName(methodName).Set(reflect.MakeFunc(field.Type, func(args []reflect.Value) (results []reflect.Value) {
			ctx := args[0].Interface().(context.Context)
			errNum := 0
			if !auth.HasPerm(ctx, defaultPerms, requiredPerm) {
				errNum++
				goto ABORT
			}
			return fn.Call(args)
		ABORT:
			err := xerrors.Errorf("missing permission to invoke '%s'", methodName)
			if errNum&1 == 1 {
				err = xerrors.Errorf("%s  (need '%s')", err, requiredPerm)
			}
			rerr := reflect.ValueOf(&err).Elem()
			if fn.Type().NumOut() == 2 {
				return []reflect.Value{
					reflect.Zero(fn.Type().Out(0)),
					rerr,
				}
			}
			return []reflect.Value{rerr}
		}))
	}
}
