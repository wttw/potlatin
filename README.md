# potlatin
Generate piglatin translations for gettext format pot files

```
Usage of potlatin:
      --html string     How to handle HTML in translations (ignore, attempt, require) (default "require")
  -o, --output string   Write output to file
```

Running `potlatin messages.pot` will generate a file `x-piglatin.po` containing translations
of the messages. If any of the source translations contain html then html tags will be passed
through untouched, while text will be translated.

Output can be directed to a file with the `--output` flag, or to stdout with `--output=-`.
