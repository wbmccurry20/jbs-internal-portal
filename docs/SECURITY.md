# Security notes

This project keeps auth and file handling focused on the basics that matter for internal operations:

- JWT-based authentication with role checks on protected routes
- Bcrypt password hashing for stored user credentials
- Request/response validation and upload size limits on the API layer
- CORS restricted to explicit origins in release mode
- Secrets kept in environment variables, not committed to git

Operational guidance:

- Keep `JWT_SECRET`, `DATABASE_URL`, and other production credentials in Railway or the local env used for deployment.
- Do not commit `.env` files or generated upload directories.
- Treat uploaded files as operational data and keep them out of source control.
- For production, configure `ALLOWED_ORIGINS` explicitly and avoid permissive wildcard CORS values.

This repo is for internal JBS operations, not public internet exposure. The goal is secure, explicit configuration and sane defaults rather than a large security rewrite.
