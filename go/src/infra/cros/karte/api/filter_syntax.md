# Filter Syntax

The filter syntax is a subset of the full language
supported by the CEL library. The syntax resembles an expression
in a generic programming language (e.g. `a.do_thing(1, "f")`).

This subset of the CEL language is compliant with AIP-160.

Karte supports a subset of this language. The subset is chosen
in such a way that it translates into a single datastore query.

This subset supports the following comparison operators:
<, <=, >, >=, ==, !=.

It also supports the connective "AND". Other connectives such as "OR"
and "NOT" are not supported.

For example, (a) and (b) are both okay, but (c) is not okay.

a) kind == "e"

b) stop_time > 1000 AND kind == "e"

c) stop_time > 1000 OR  kind == "e" 

For a list of possible fields that are supported by each List method,
check the documentation of the respective method.
