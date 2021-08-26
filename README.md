the very specialized static site generator behind 
[octothorpean.org](https://www.octothorpean.org/)

It depends on the `chevron` mustache-like template
library. Requirements look something like
`chevron==0.14.0`.

Web page style/JS is built with
[Bootstrap](https://getbootstrap.com/docs/3.4/css/).

This spoiler-free repo doesn't contain puzzle content.
It's just a static site generator that slurps in puzzle
content in a directory on my home machine's hard drive
and generates site pages in another directory on my
home machine's hard drive. 

"Deploy" is `scp`.

Instead of a static site, Octothorpean used to be
a Google App Engine app; don't be surprised if you
bump into old info that thinks that's still true.
