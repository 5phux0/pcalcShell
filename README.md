pcalcShell
==========

A shell calculatior based on the PowerCalc library

Current features:
  Interperet entered expression strings with the correct order of operation
  Easy access to common constants (well six of them this far)
  Create and combine functions containing unknown variables and evaluate them with different arguments
  Calculations return values with the correct order of significant digits !!!!BUG!!!!
  
Planed features:
  Graph drawing and perhaps some other types of analysis features
  Read values written on scientific notation
  Handle lists and sets of values
  
Bugs:
  Floats are sometimes printed with the wrong number of significant digits.
  Expressions called with arguments containing parenthesis enclosed ',' wont parse correctly
