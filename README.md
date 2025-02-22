# Scaffold

Scaffold is a cli that allows Developer to easily scaffold (create or update files) according to best practices.

## Usage

```bash
 scaffold
> Pick a template
> Fill required information for the template
> Profit
```

Optionally if you know what you're looking for:

```bash
 scaffold externalhttp # --package app as an example
```

Scaffold allows a wide variety of formatting options, such as template defined inputs, where the files should be placed, if they should be overwritten or not.

## Develop your own template

Templates are maintained in the `registry` folder. This is automatically kept up-to-date by the `scaffold`, in the folder you will see all the available templates.

To develop your own

```
go run ./main.go --registry registry scaffold --name your_scaffold_here
```

Scaffold will now have created a sample scaffold in the `registry/your_scaffold_here` folder along with tests.

A template consists of the following files:

- `scaffold.yaml`: Controls how the scaffold is supposed to work, which inputs it has, etc.
- `scaffold_test.go`: Optional, but recommended tests which runs a set of input on the template and checks the output
- `files/*.gotmpl`: Files to be scaffolded, it is recommended to provide a suffix of `.gotmpl` to the files, especially for golang files, which might otherwise mess with the project. Each file is templated using go templates, and is put in the path specified by the user, or via. the default path from the scaffold file. Files can also be put in directories, and the folder structure will be preserved.
- `testdata/your_test_here/actual && expected`: contains tests to match the output of the scaffolder. This is especially useful to test a variety of input, for example using the defaults, vs. getting other info from the user.

To make your plugin available, simply create a pr on this repository, merge and your users should have it available shortly
