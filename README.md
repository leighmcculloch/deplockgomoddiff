# deplockgomoddiff

A tool that compares [dep](https://github.com/golang/dep) `Gopkg.lock` and
[Modules](https://github.com/golang/go/wiki/Modules) `go list -m all` output to
validate differences after migrating from dep to Modules.

## Install
```
go get github.com/leighmcculloch/deplockgomoddiff
```

## Usage
```
Usage of deplockgomoddiff:
  -d string
        dep Gopkg.lock file
  -m string
        file containing the output of 'go list -m all'
  -p string
        password or personal access token to auth with GitHub (optional)
  -u string
        username to auth with GitHub (optional)
  -help
        print this help
```

## Example
```
deplockgomoddiff -d Gopkg.lock -m <(go list -m all)
```

## Notes
The tool will make HTTP calls to the GitHub API to retrieve tags and revisions
for mismatching dependencies to attempt to confirm if dependencies have changed
version or are just using a different tag name that points to the same
revision. The tool won't use any authentication by default which may trigger
GitHub's rate limiting. Add the username and password or personal access token
if rate limiting occurs.

Migrating from dep to Modules can be challenging if your repository is large or
has complicated dependencies. The Go blog has a great tutorial about migrating
to Modules from other dependency managers:
https://blog.golang.org/migrating-to-go-modules. 
