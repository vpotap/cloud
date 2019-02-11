# go-harbor

A Harbor API client enabling Go programs to interact with Harbor in a simple and uniform way

[![GitHub license](https://img.shields.io/github/license/ygqbasic/go-harbor.svg?style=flat)](https://github.com/ygqbasic/go-harbor/blob/master/LICENSE)
![GitHub repo-size](https://img.shields.io/github/repo-size/ygqbasic/go-harbor.svg?style=flat)
![GitHub release-date](https://img.shields.io/github/release-date-pre/ygqbasic/go-harbor.svg?style=flat)
![GitHub release-pre](https://img.shields.io/github/release-pre/ygqbasic/go-harbor.svg?style=flat)
![GitHub contributors](https://img.shields.io/github/contributors/ygqbasic/go-harbor.svg?style=flat)
## 开始
## Coverage

This API client package covers most of the existing Harbor API calls and is updated regularly
to add new and/or missing endpoints. Currently the following services are supported:

- [ ] Users
- [x] Projects
- [x] Repositories
- [ ] Jobs
- [ ] Policies
- [ ] Targets
- [ ] SystemInfo
- [ ] LDAP
- [ ] Configurations

## Usage

```go
import "github.com/ygqbasic/go-harbor"
```

Construct a new Harbor client, then use the various services on the client to
access different parts of the Harbor API. For example, to list all
users:

```go
harborClient := harbor.NewClient(nil, "url","username","password")
statisticMap, _, errs := harborClient.GetStatistics()
```

Some API methods have optional parameters that can be passed. For example,
to list all projects for user "haobor":

```go
harborClient := harbor.NewClient(nil, "url","username","password")
opt := &ListProjectsOptions{Name: "haobor"}
projects, _, err := harborClient.Projects.ListProject(opt)
```

For complete usage of go-harbor, see the full [package docs](https://godoc.org/github.com/ygqbasic/go-harbor).

## ToDo

- The biggest thing this package still needs is tests :disappointed:

## Issues

- If you have an issue: report it on the [issue tracker](https://github.com/ygqbasic/go-harbor/issues)
