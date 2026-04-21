// Package code implements a coding agent that drives an LLM conversation
// loop to implement plan steps against a sandboxed project directory using
// file-system and shell tools.
//
// Class: platform
// UseWhen: Building a coding agent (code-hand). Implements Implement(projectDir, step, feedback). Not for orchestrators or non-coding workers.
package code
