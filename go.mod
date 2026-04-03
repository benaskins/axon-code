module github.com/benaskins/axon-code

go 1.26.1

require (
	github.com/benaskins/axon-loop v0.0.0-00010101000000-000000000000
	github.com/benaskins/axon-talk v0.8.1
	github.com/benaskins/axon-tool v0.3.0
)

require github.com/benaskins/axon-tape v0.1.1 // indirect

replace (
	github.com/benaskins/axon-loop => /Users/benaskins/dev/lamina/axon-loop
	github.com/benaskins/axon-talk => /Users/benaskins/dev/lamina/axon-talk
	github.com/benaskins/axon-tool => /Users/benaskins/dev/lamina/axon-tool
)
