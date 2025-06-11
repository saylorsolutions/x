/*
Package assert provides runtime assertion support for expressing complex validation constraints in a user-friendly way.

There are a few patterns that are supported:
  - Collecting many possible errors into one.
  - Assertions that panic if they are violated.
  - Removal of assertions with a build flag to maintain runtime performance.

To turn off assertions build with the 'noassert' flag.
For temporary changes, the Disable and Enable functions are also provided, but these should likely not be used in production code.
*/
package assert
