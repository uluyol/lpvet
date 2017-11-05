# lpvet

CPLEX is useful for solving optimization problems.
Unfortunately, debugging bad problem formulations can be a pain.
lpvet tried to make this easier by checking CPLEX lp formulations for common mistakes.
At the moment, lpvet only looks for errors involving misuse of variables:
errors are reported for undefined variables
warnings are reported for unused variables.
By default, only errors are shown.
