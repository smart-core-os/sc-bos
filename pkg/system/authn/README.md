## Local Authentication

The controller may use this system to optionally accept local authentication - i.e. a username and password. This
method of authentication is less secure but useful when the area controller is running without any supporting
infrastructure, like a dedicated OAuth server.

Local authentication uses [OAuth2 password grant](https://www.oauth.com/oauth2-servers/access-tokens/password-grant/) to
exchange a username and password for a trusted token.

To setup local authentication you need to do two things:

1. Let the area controller know about the local accounts you wish to allow
2. Turn on the local password-based OAuth server

For step 1 we need to create a `users.json` file in the area controllers data directory (defaults to
`.data/area-controller-01/users.json`). The structure of this file, like `tenants.json`, is a JSON list of accounts and
their hashed passwords:

```json
[
  {
    "id": "email@example.com",
    "title": "My Name",
    "secrets": [
      {"hash": "$2a$10$/uBhiEncrKMgJ8q5AjyRFuqe1dzTNTsOjX1noIzu/lI5JQ78EUvLO"}
    ],
    "zones": ["Floor1", "Floor2"]
  }
]
```

The secret hash can be generated using the locally provided `pash` tool:

```shell
go run github.com/smart-core-os/sc-bos/cmd/pash           
Password: <enter your password>
$2a$10$/uBhiEncrKMgJ8q5AjyRFuqe1dzTNTsOjX1noIzu/lI5JQ78EUvLO
```

Finally we can turn on the OAuth password flow for the area controller via this `authn` system config

```json5
// .data/area-controller-01/system.json
{
  "systems": {
    "authn": {
      "user": {
        "fileAccounts": true
      }
    }
  }
}
```
