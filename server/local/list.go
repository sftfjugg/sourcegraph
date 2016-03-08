// GENERATED CODE - DO NOT EDIT!
//
// Generated by:
//
//   go run gen_list.go -o list.go
//
// Called via:
//
//   go generate
//

package local

import (
	"src.sourcegraph.com/sourcegraph/svc"
)

// Services contains all services implemented in this package.
var Services = svc.Services{
	Accounts:          Accounts,
	Annotations:       Annotations,
	Auth:              Auth,
	Builds:            Builds,
	Defs:              Defs,
	Deltas:            Deltas,
	GitTransport:      GitTransport,
	GraphUplink:       GraphUplink,
	Meta:              Meta,
	MirrorRepos:       MirrorRepos,
	MultiRepoImporter: Graph,
	Notify:            Notify,
	Orgs:              Orgs,
	People:            People,
	RegisteredClients: RegisteredClients,
	RepoStatuses:      RepoStatuses,
	RepoTree:          RepoTree,
	Repos:             Repos,
	Search:            Search,
	Storage:           Storage,
	Users:             Users,
}
