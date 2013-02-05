# HTTP MultiPoster

This is a tool to help reliably deliver a webhook to multiple
destinations.  The given input will be provided to each of the
specified URLs concurrently until all of them have succeeded or it has
tried too many times.

Example:

    ./multipost http://something/ http://somethingelse/ < /some/input
