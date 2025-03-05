# Scaffold

Scaffold is a cli that allows Developer to easily scaffold (create or update files) according to best practices.

![demo](assets/demo.gif)

## Install

### Brew

```bash
brew install kjuulh/tap/scaffold 
```

### Go

```bash
go install github.com/kjuulh/scaffold@latest 
```

## Usage

```bash
# set this in your .zshrc or .bashrc file
export SCAFFOLD_REGISTRY=https://github.com/kjuulh/scaffold.git # you can use your own templates as well, see the Develop your own template section

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

You can of course use the default scaffold maintained here, but to create your own, simply fork the [example registry](https://github.com/kjuulh/scaffold-example-registry).

To develop your own

```
./scaffold.sh scaffold --name <your new template name here>
```

Scaffold will now have created a sample scaffold in the `registry/your_scaffold_here` folder along with tests.

A template consists of the following files:

- `scaffold.yaml`: Controls how the scaffold is supposed to work, which inputs it has, etc.
- `scaffold_test.go`: Optional, but recommended tests which runs a set of input on the template and checks the output
- `files/*.gotmpl`: Files to be scaffolded, it is recommended to provide a suffix of `.gotmpl` to the files, especially for golang files, which might otherwise mess with the project. Each file is templated using go templates, and is put in the path specified by the user, or via. the default path from the scaffold file. Files can also be put in directories, and the folder structure will be preserved.
- `testdata/your_test_here/actual && expected`: contains tests to match the output of the scaffolder. This is especially useful to test a variety of input, for example using the defaults, vs. getting other info from the user.

To make your plugin available, simply create a pr on this repository, merge and your users should have it available shortly

### Bonus testing

![test demo](./assets/test-demo.gif)

You'll notice that there is a _test file, as well as a testdata folder.

These are not mandatory, but allows you to test a variety of inputs for your template, the tests are snapshot based tests. In the testdata folder you'll see an actual folder as well as expected. To accept the output of a test, simply delete the expected if found, and rename the actual to expected.
