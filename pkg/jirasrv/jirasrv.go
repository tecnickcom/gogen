/*
Package jirasrv provides a lightweight, typed HTTP client foundation for Jira
Server REST integrations.

# Problem

Integrating with Jira Server typically requires repeating the same low-level
HTTP plumbing: base URL composition, bearer-token headers, JSON encoding,
request validation, retry policy setup, and health checks. This boilerplate can

	obscure domain logic and increase inconsistency across services.

# Solution

This package wraps those concerns in a reusable [Client] while exposing a
generic request entry point ([Client.SendRequest]) and typed Jira data models
for common payloads (issues, transitions, comments, users, visibility,
properties, metadata).

Core responsibilities handled by the client:
  - API base-path composition (`/rest/api/2`)
  - bearer-token authentication headers
  - JSON request encoding and content headers
  - request payload validation
  - method-aware HTTP retry behavior
  - connection and service health probing via [Client.HealthCheck]

# Extensibility

The client supports dependency injection and runtime tuning via options:
  - custom HTTP transport/client ([WithHTTPClient])
  - request and ping timeouts ([WithTimeout], [WithPingTimeout])
  - retry policy controls ([WithRetryAttempts], [WithRetryDelay])

This design makes the package suitable both for production use and deterministic
unit testing with mocked HTTP clients.

# Reference

Jira Server REST API reference:
  - https://docs.atlassian.com/software/jira/docs/api/REST/9.17.0/

# Benefits

jirasrv centralizes Jira HTTP integration concerns into a concise, testable
client layer, reducing repetitive infrastructure code and enabling teams to
focus on Jira domain workflows.
*/
package jirasrv
