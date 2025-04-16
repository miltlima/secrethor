# Changelog

{{ if .Versions -}}
{{ range .Versions }}
## {{ if .Tag.Previous }}[{{ .Tag.Name }}]({{ $.Info.RepositoryURL }}/compare/{{ .Tag.Previous.Name }}...{{ .Tag.Name }}){{ else }}{{ .Tag.Name }}{{ end }}

{{ range .CommitGroups -}}
### {{ .Title }}

{{ range .Commits -}}
* {{ if .Scope }}**{{ .Scope }}:** {{ end }}{{ .Subject }}
{{- if .Refs }} {{ range .Refs }}([#{{ .Ref }}]({{ $.Info.RepositoryURL }}/issues/{{ .Ref }})){{ end }}{{ end }}
{{ end }}
{{ end -}}

{{- if .RevertCommits -}}
### Reverts

{{ range .RevertCommits -}}
* {{ .Revert.Header }}
{{ end }}
{{ end -}}

{{- if .MergeCommits -}}
### Pull Requests

{{ range .MergeCommits -}}
* {{ .Header }}
{{ end }}
{{ end -}}

{{- if .NoteGroups -}}
{{ range .NoteGroups -}}
### {{ .Title }}

{{ range .Notes }}
{{ .Body }}
{{ end }}
{{ end -}}
{{ end -}}
{{ end -}}

{{ else }}
## Initial Release

* First version of Secrethor
* Features include:
  * Secret scanning in Kubernetes clusters
  * Orphaned secrets detection
  * Comprehensive security analysis
  * Multiple output format support
{{ end -}}