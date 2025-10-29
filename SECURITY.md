# Security Policy

## Supported Versions

We provide security updates for the following versions:

| Version | Supported          |
| ------- | ------------------ |
| 1.0.x   | :white_check_mark: |
| < 1.0   | :x:                |

## Reporting a Vulnerability

If you discover a security vulnerability, please report it privately. **Do not open a public GitHub issue.**

### How to Report

1. **Email**: Send details to security@temabit.com (or your security contact)
2. **Include**:
   - Description of the vulnerability
   - Steps to reproduce
   - Potential impact
   - Suggested fix (if available)

### Response Process

1. You will receive an acknowledgment within 48 hours
2. We will investigate and provide an initial assessment within 5 business days
3. We will keep you informed of our progress
4. Once fixed, we will release a security update and credit you (if desired)

### Disclosure Policy

- We follow responsible disclosure practices
- Vulnerabilities will be disclosed after a fix is available
- Updates will be released as soon as possible

## Security Best Practices

When using this provider:

1. **Protect Credentials**:
   - Never commit API tokens to version control
   - Use environment variables or Terraform Cloud/Enterprise secrets
   - Rotate API tokens regularly

2. **Least Privilege**:
   - Use API tokens with minimal required permissions
   - Regularly review and audit access

3. **Network Security**:
   - Use HTTPS endpoints only
   - Verify TLS certificates in production

4. **Version Management**:
   - Keep the provider updated to the latest version
   - Review changelogs for security updates

## Known Security Considerations

- API tokens are transmitted using Basic Authentication (Base64 encoded, not encrypted)
- Always use HTTPS connections (provider enforces this)
- API tokens provide access to all Compass components you have access to

