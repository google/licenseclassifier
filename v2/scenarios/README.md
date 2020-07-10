# Scenarios

Test scenarios are files with a predefined structure for running tests.

A scenario should have some text describing what is being tested and why.
Following the description, the keyword EXPECTED followed by a colon and a
comma-separated list of expected licenses should appear, concluding with an
endline. All remaining content in the file is passed as input to the classifier.

Filenames are generally the Buganizer id, suffixed with an underscore and
numbers if there are multiple scenarios necessary for a bug.

As an example:

```
This is a simple license header that we did not previously detect due
to b/12345
EXPECTED:ISC
<code content goes here>
```

Scenarios should not be encumbered by license restrictions, so it's essential to
create a minimal reproduction that doesn't rely on licensed code.
