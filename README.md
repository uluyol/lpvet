# lpvet

CPLEX is useful for solving optimization problems.
Unfortunately, debugging bad problem formulations can be a pain.
lpvet tried to make this easier by checking CPLEX lp formulations for common mistakes.

At the moment, lpvet only looks for errors involving misuse of variables:
errors are reported for undefined variables
warnings are reported for unused variables.

By default, only errors are shown.

To handle regular, continuous variables, lpvet requires users to add a special section called CONTINUOUS.
Because this is specfic to lpvet, you will need to insert these as comments with the lpvet: prefix with nothing inbetween the \ and lpvet:.

For example, if a, b, and c are continuous variables, add

```
\lpvet:CONTINUOUS
\lpvet:    a
\lpvet:    b
\lpvet:	   c
```
