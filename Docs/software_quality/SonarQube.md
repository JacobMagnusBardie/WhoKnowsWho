### SonarQube

Static code analyser that scans code for security and maintainability issues.

https://sonarcloud.io/organizations/jacobmagnusbardie/

*After you have setup (a) code quality tool(s) and gone through the issues, your group should create a brief document that answers the following questions:*

## Found issues

- login.html -> logins have no id or '<label>'

- integration_test.go ->  complexity 21 (limit 15) "complexity is measured by counting execution paths & how hard code is to read"

- main.go -> "layout.html" repeated 3 times
- main.go -> "Content-Type" repeated 8 times
- main.go ->  "application/json" repeated 5 times.
- main_test.go -> complexity 16 (limit 15)
- style.css -> footer text contrast too low

Do you agree with the findings?

- Which ones did you fix?

-> login.html

- Which ones did you ignore?

-> main_test.go

-> integration_test.go

-> main.go (layout.html, Content-Type, application/json)

-> style.css

- Why?

Why was it not corrected

-> main_test.go becuase complexity overstepped by 1 (16/15) is not an issue for a project of our respective size. Also even if larger project one could argue allowing overstepping by 1, as these complexities are guidelines/agreed conventions and 16 rather than 15 barely makes a difference, practically. 

-> integration_test.go because it is test code, not production code, and it walks through the whole auth flow step by step, so it is long by nature. Complexity 21 (limit 15) is not great, but splitting it up would mean repeating the same setup in several tests, which is not worth it for a project of our size. Not urgent, because it does not affect users or how the app behaves, only how easy the test is to read.

-> main.go (layout.html, Content-Type, application/json) because the repeated literals are all in one file and easy to read. A typo in "layout.html" would crash the app at startup, so it can't go unnoticed, and "Content-Type"/"application/json" repeated 8 and 5 times is guidelines/agreed conventions rather than a bug. Constants would be a nice cleanup, but not important right now, and could be done if the file grows.

-> style.css because it is only the footer text (copyright and About link), and the low contrast only affects readability a bit. It will be corrected later, as part of a larger, incoming frontend improvement. Not a specific focus now. 

Why was it corrected

login.html because lack of id's or label means a screenreader cannot identify it. (no compatibility for people who cannot read)