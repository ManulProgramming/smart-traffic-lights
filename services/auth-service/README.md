# Auth Service

Go authentication/authorization service using PostgreSQL, pgx, Argon2id, JWT access tokens, and server-side sessions.

## Architecture

```text
HTTP -> middleware -> handler/controller -> service -> repository -> PostgreSQL
```

### Implemented endpoints
```text
POST   /api/v1/register
POST   /api/v1/login
POST   /api/v1/logout
GET    /api/v1/validate

GET    /api/v1/users
GET    /api/v1/user/{id}
PATCH  /api/v1/user/{id}
DELETE /api/v1/user/{id}
```

### Registration

multipart/form-data:
```
name
email
password
optional picture
```

Pictures are validated as actual JPEG data, limited to 1 MiB, and must be exactly 50×50 pixels. User, picture, and session creation happen in the same database transaction.

### Login

JSON:
```json
{
  "name": "alice",
  "password": "correct horse"
}
```
or:
```json
{
  "email": "alice@example.com",
  "password": "correct horse"
}
```

### Update

`PATCH /api/v1/user/{id}` requires the user's Bearer token and current password.

JSON fields:

```json
{
  "name": "new_name",
  "email": "new@example.com",
  "new_password": "new password",
  "current_password": "old password"
}
```

Multipart requests may additionally contain a picture field.

Changing a password revokes all active sessions.

### Delete

`DELETE /api/v1/user/{id}` requires the matching user's Bearer token and:

```json
{
  "password": "current password"
}
```

The user deletion cascades to pictures and sessions.

### Authorization

Regular users can read their own user record and modify/delete their own account. ADMIN can read any user. The role schema remains the source of authorization policy rather than trusting arbitrary client-supplied claims.

Error format
```json
{
  "status": 400,
  "description": "invalid email format",
  "date": "2026-09-13T20:00:00Z",
  "code": "INVALID_EMAIL",
  "request_id": "..."
}
```

## Security features

- Argon2id password hashes with per-password random salts.
- HS256 JWTs with explicit algorithm validation, issuer, audience, expiry, and not-before checks.
- Server-side session records allow logout/revocation.
- JWTs contain a session ID and are hashed before persistence.
- Password changes revoke all sessions.
- Authentication failures do not reveal whether a user exists.
- Request-size limits.
- JPEG content and dimensions are validated server-side.
- Request IDs and structured request logging.
- Panic recovery.
- Basic security response headers.
- Pagination on user listing.
- Database foreign keys with cascading cleanup.

## Run

Copy .env.template to .env, set a real JWT_SECRET and DATABASE_URL.

Create a docker image:
```bash
docker build -f Dockerfile -t <some-tag> .
```

Deployment:
WIP

## Testing

Run:

```bash
go test ./...
```