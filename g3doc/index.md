# License Classifier

go/license-classifier

[TOC]

TODO: The license classifier currently resides in `//third_party/golang`, but
that should change since it doesn't use `//third_party/golang/update` for
updating.

## Introduction

The license classifier is a library and set of tools that can analyze text to
determine what type of license it contains. It searches for license texts in a
file and compares them to an [archive of known
licenses](http://google3/third_party/golang/licenseclassifier/licenses). A file
could contain one or more licenses. The classifier can also analyze source code
files for license texts in a comment.

A "confidence level" is associated with each result indicating how close the
match was. A confidence level of `1.0` indicates an exact match, while a
confidence level of `0.0` indicates that no license was able to match the text.

## Design doc

[Original Design Doc](http://go/license-classifier)

## GitHub repository

https://github.com/google/licenseclassifier

## Buganizer component:

[Open Source > Compliance > License
Classifier](http://b/issues/new?component=317883)


## Release process

The license classifier is used as a library. It's main development takes places
in the GitHub repository and then Copybara is used to sync it.

The `identify_license` tool is released to `x20` [via
Rapid](http://rapid/project/identify_license). It's installed at
`/google/data/ro/teams/opensource/identify_license`.

## Adding a new license

Adding a new license is straight forward:

1.  Create a file in google3/third_party/golang/licenseclassifier/licenses.

    *   The filename should be the name of the license or its abbreviation. If
        the license is an Open Source license, use the appropriate identifier
        specified at https://spdx.org/licenses/.
    *   If the license is the "header" version of the license, append the suffix
        "`.header`" to it. See
        google3/third_party/golang/licenseclassifier/licenses/README.md more
        details.

2.  Add the license name to the list in `license_type.go`.

3.  Regenerate the `licenses/licenses.db` file by running the license
    serializer:

    ```shell
    $ license_serializer -output $LICENSE_CLASSIFIER_DIR/licenses
    ```

4.  Create and run appropriate tests to verify that the license is indeed
    present.

## Tools

### Identify license

`identify_license` is a command line tool that can identify the license(s)
within a file.

```shell
$ identify_license LICENSE
LICENSE: GPL-2.0 (confidence: 1, offset: 0, extent: 14794)
LICENSE: LGPL-2.1 (confidence: 1, offset: 18366, extent: 23829)
LICENSE: MIT (confidence: 1, offset: 17255, extent: 1059)
```

### License serializer

The `license_serializer` tool regenerates the `licenses/licenses.db` archive.
The archive contains preprocessed license texts for quicker comparisons against
unknown texts.

```shell
$ license_serializer -output licenseclassifier/licenses
```
