package rbac

import (
	"context"
	"fmt"

	"github.com/zzztttkkk/suna/sqls"
	"github.com/zzztttkkk/suna/utils"
)

type permOpT struct {
	sqls.EnumOperator
}

var _PermissionOperator = &permOpT{}

func init() {
	dig.Provide(
		func() _DigPermissionTableInited {
			_PermissionOperator.Init(
				Permission{},
				func() sqls.EnumItem { return &Permission{} },
				nil,
			)
			return _DigPermissionTableInited(0)
		},
	)
}

func (op *permOpT) Create(ctx context.Context, name, descp string) {
	defer LogOperator.Create(ctx, "perm.create", utils.M{"name": name, "descp": descp})
	op.EnumOperator.Create(ctx, name, descp)
}

func (op *permOpT) Delete(ctx context.Context, name string) {
	defer LogOperator.Create(ctx, "perm.delete", utils.M{"Name": name})
	op.EnumOperator.Delete(ctx, name)
}

func (op *permOpT) List(ctx context.Context) (lst []*Permission) {
	for _, enum := range op.EnumOperator.List(ctx) {
		lst = append(lst, enum.(*Permission))
	}
	return
}

func EnsurePermission(name, descp string) string {
	if _PermissionOperator.ExistsByName(context.Background(), name) {
		return name
	}
	tcx, committer := sqls.Tx(context.Background())
	defer committer()
	_PermissionOperator.EnumOperator.Create(tcx, name, fmt.Sprintf("%s; created by `EnusrePermission`", descp))
	return name
}
