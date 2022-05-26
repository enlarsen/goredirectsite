# Using

```bash
cd goredirectsite
go run . <base-url-for-new-site> <default-page-id> <old-site-md-files> <new-site-md-files> <redir-site-output-directory>
```

| Name | Description |
|------|-------------|
| base-url-for-new-site | The url on the destination site that is used to generate all of the redirects. It's used to form destination URLs: base-url-for-new-site concatenated with the destination pages's ID |
| default-page-id | 
| old-site-md-files | The on disk location of the old site's .md files. Should be the base directory for all of the Markdown files |
| new-site-md-files | The on disk location of the new site's .md files. The YAML ID is extracted from these files and concatenated with base-url-for-new-site to form destination page URLs. |
| redir-site-output-directory | Where goredirectsite writes its redirect files |


## Example for axe-devtools-html

Create a redirect site at ~/Desktop/test-site

```bash
go run . "https://docs.deque.com/devtools-html/4.0.0/en" ~/src/HTML-docs-website/content/ ~/src/docs-devtools-html/content/4.0/en/ ~/Desktop/test-site
```

## Example for axe-linter

TBD
