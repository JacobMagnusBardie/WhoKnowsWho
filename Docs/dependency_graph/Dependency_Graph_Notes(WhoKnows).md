**1st layer -> Makefile (Entry Points)**
The Makefile is a saved shell command with a short name attached.
Runs the shell commands as provided in their sequential order. 
Together, these commands run the program. 

**Base layer -> Application code (app\_py, app\_test)**
This is the "Hub" - most things are connected to it. 
app\_py depends on flask, python2(library imports etc)
and all of the templates, in order to run. 

**Data layer -> (schema\_sql, db\_file)**
schema\_sql is a build-time dependency and depends on db\_file in order to function. 
app\_py also depends on db\_file in order to properly fetch during run time. 

**Template layer -> (all of the html files, ie. search\_html, about\_html etc)**
all depend on layout\_html in order to function, which functions as a base layout

