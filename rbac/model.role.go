package rbac

import (
	"github.com/jmoiron/sqlx"
	"github.com/zzztttkkk/suna/sqls"
)

type roleT struct {
	sqls.Enum
	Descp string `json:"descp"`

	Based       []int64 `json:"based" db:"-"`
	Permissions []int64 `json:"permissions" db:"-"`
}

func (roleT) SqlsTableName() string {
	return tablePrefix + "role"
}

func (role roleT) SqlsTableColumns(db *sqlx.DB) []string {
	return role.Enum.SqlsTableColumns(db, "descp text")
}

type roleInheritanceT struct {
	Role  int64 `json:"role"`
	Based int64 `json:"based"`
}

func (roleInheritanceT) SqlsTableName() string {
	return tablePrefix + "role_inheritance"
}

func (ele roleInheritanceT) SqlsTableColumns() []string {
	return []string{
		"role bigint not null",
		"based bigint not null",
		"primary key(role, based)",
	}
}

type roleWithPermT struct {
	Role int64 `json:"role"`
	Perm int64 `json:"perm"`
}

func (roleWithPermT) SqlsTableName() string {
	return tablePrefix + "role_with_perm"
}

func (ele roleWithPermT) SqlsTableColumns() []string {
	return []string{
		"role bigint not null",
		"perm bigint not null",
		"primary key(role, perm)",
	}
}
