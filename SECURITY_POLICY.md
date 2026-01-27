# Security Policy

## Reporting a Vulnerability

If you discover a security vulnerability in the JBS Internal Portal, please report it to:

**Email**: hello@itwill.dev

Please include:
- Description of the vulnerability
- Steps to reproduce
- Potential impact
- Suggested fix (if any)

**Do not** open public GitHub issues for security vulnerabilities.

## Response Time

- We will acknowledge receipt within 24 hours
- We will provide an initial assessment within 72 hours
- We will work on a fix and notify you when it's deployed

## Supported Versions

| Version | Supported          |
| ------- | ------------------ |
| 1.x.x   | :white_check_mark: |

## Security Measures

### Authentication & Authorization
- JWT tokens with bcrypt password hashing
- Rate limiting on authentication endpoints
- Secure password requirements

### Data Protection
- HTTPS enforced in production
- SQL injection prevention via parameterized queries
- File upload validation and sanitization
- CORS protection

### Network Security
- Security headers (CSP, X-Frame-Options, etc.)
- Rate limiting on all endpoints
- Request logging for audit trails

### Infrastructure
- Regular dependency updates
- Automated security scanning
- Environment-based secrets management

## Best Practices for Deployment

1. **Change default passwords** immediately after deployment
2. **Rotate JWT_SECRET** regularly (at least every 90 days)
3. **Enable HTTPS only** - Never serve over HTTP
4. **Limit ALLOWED_ORIGINS** to specific domains
5. **Monitor logs** for suspicious activity
6. **Keep dependencies updated** regularly
7. **Use strong SUPPORT_PASSWORD** - never use defaults

## Security Checklist for Production

- [ ] All default passwords changed
- [ ] JWT_SECRET is a strong, random value
- [ ] ALLOWED_ORIGINS limited to production domains
- [ ] HTTPS enabled and enforced
- [ ] Database uses SSL connection
- [ ] File upload directory has proper permissions
- [ ] Regular backups configured
- [ ] Monitoring and alerting set up
- [ ] Rate limiting configured appropriately
- [ ] Security headers enabled
