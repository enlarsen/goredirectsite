# Using

```bash
cd goredirectsite
go run . <base-url-for-new-site> <old-site-md-files> <new-site-md-files> <redir-site-output-directory>
```

## Example for axe-devtools-html

Create a redirect site at ~/Desktop/test-site

```bash
go run . "https://docs.deque.com/devtools-html/4.0.0/en" ~/src/HTML-docs-website/content/ ~/src/docs-devtools-html/content/4.0/en/ ~/Desktop/test-site
```

## Example for axe-linter

TBD
