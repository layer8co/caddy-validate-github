# caddy-validate-github

Caddy v2 middleware for validating a GitHub webhook request.

## Installation

```
xcaddy build \
    --with github.com/layer8co/caddy-validate-github
```

## Usage

This middleware functions as a gatekeeper for any succeeding directives in a
route.  The request is only passed on to the next directive if its signature and
(optionally) conditions are valid. Otherwise, the client receives a 403 status code.

### Caddyfile

The plugin has two syntaxes for caddyFiles. Pick up the one which is more appropriate for you.

#### Simple syntax

Simply validate the secret and let it pass:

```
validate_github_webhook <secret>
```
* **secret** - shared secret between you and GitHub


> [!TIP]
> You could use a tool like `openssl` to generate a secure secret: `openssl rand
> -hex 12`

#### Nested Syntax

In the nested syntax, you can also define some matching conditions that gets
validated with the supplied payload from Github:

```
validate_github_webhook {
  secret <secret>

  match {
    action completed
    workflow_run.name "Create and deploy"
    workflow_run.status completed
    workflow_run.conclusion success
  }
}
```

> [!TIP]
> Check out
> [here](https://docs.github.com/en/webhooks/webhook-events-and-payloads) to
> learn more about what's available in the github's webhook payload.

#### Example

`validate_github_webhook` is a middleware meant to precede other directives.

An example of this directive in context looks like this:

```
route /update {
    validate_github_webhook KcuP9N0iEqYHFBRUda6oHLP4UUub6EMz
    exec * /path/to/bin/deploy.sh
}
```

Here, you're using `validate_github_webhook` to validate the request before passing
it along to [caddy-exec](https://github.com/abiosoft/caddy-exec), which runs the
`/path/to/bin/deploy.sh` script. Since `caddy-exec` does not support chaining
commands at this time, it's necessary to perform multiple commands in a script
or Go binary and invoke it from the `exec` directive. 

By using the module in this way, it can act as a simple [continues-deployment solution](https://woodbeck.net/posts/validate-github-webhook-middleware).

### JSON

The `validate_github_webhook` JSON look like this, minus succeeding middleware:
```json
{
  "routes": [
    {
      "handle": [
        {
          "handler": "validate_github_webhook",
          "secret": "KcuP9N0iEqYHFBRUda6oHLP4UUub6EMz"
        }
      ],
      "match": [
        {
          "path": [
            "/refresh"
          ]
        }
      ]
    }
  ]
}
```

## Credits

Main credit goes to [Adam
woodbeck](https://github.com/awoodbeck/caddy-validate-github), the original
creator of the module.

