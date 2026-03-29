/*
Package slack provides a lightweight client for sending messages to Slack via
Incoming Webhooks.

Slack webhook reference:
https://api.slack.com/messaging/webhooks

# Problem

Posting to Slack webhooks directly from raw HTTP code introduces repeated
boilerplate: JSON payload shaping, timeout handling, retries, optional sender
metadata defaults (username/icon/channel), and status checks. Repeating that
in each service increases integration drift and failure-handling inconsistencies.

This package centralizes those concerns behind a small client API.

# What It Provides

  - [New] creates a configured webhook client.
  - [Client.Send] sends a text message with per-message overrides for username,
    icon emoji, icon URL, and channel, while falling back to client defaults.
  - [Client.HealthCheck] performs an availability check against a status
    endpoint (default: Slack Status API) with a dedicated ping timeout.

# Key Features

  - Simple send API for webhook messages with sensible defaults.
  - Optional per-message metadata overrides without reconstructing the client.
  - Write-request retry strategy (via gogen httpretrier) for transient failures.
  - Configurable HTTP behavior through options:
    [WithTimeout], [WithPingTimeout], [WithPingURL],
    [WithHTTPClient], [WithRetryAttempts].
  - Health checks suitable for readiness/liveness integrations.

# Usage

	c, err := slack.New(
	    webhookURL,
	    "my-bot",
	    ":rocket:",
	    "",
	    "#deployments",
	)
	if err != nil {
	    return err
	}

	if err := c.HealthCheck(ctx); err != nil {
	    return err
	}

	err = c.Send(ctx,
	    "deployment succeeded",
	    "", // use default username
	    "", // use default icon emoji
	    "", // use default icon URL
	    "", // use default channel
	)
	if err != nil {
	    return err
	}

This package is ideal for Go services that need a minimal, dependable Slack
notification integration without hand-writing webhook request plumbing.
*/
package slack
