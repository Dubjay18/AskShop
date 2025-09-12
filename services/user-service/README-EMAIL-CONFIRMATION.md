# Email Confirmation with Supabase

This document provides guidance on setting up email confirmation with Supabase in your AskShop application.

## Background

When users register in Supabase, by default they receive an email with a confirmation link. The user needs to click this link to verify their email address before they can log in. This is a security feature to prevent spam and ensure users provide valid emails.

## Configuration Options

### 1. Setting a Redirect URL

When Supabase sends verification emails, you need to configure where users should be redirected after clicking the verification link. Set the following environment variable:

```
SUPABASE_REDIRECT_URL=https://your-app-domain.com/auth/callback
```

This URL should be a page in your application that can handle the authentication flow after email confirmation.

### 2. Automatic Email Confirmation

For development or testing environments where email delivery might not be set up, you can use our automatic email confirmation feature. The application will:

1. Register the user normally
2. Detect when login fails due to "email_not_confirmed" error
3. Automatically confirm the email using the admin API
4. Retry the login

This approach works in development but should be used carefully in production.

## How It Works

1. When a user registers, if `SUPABASE_REDIRECT_URL` is set, it will be included in the signup request.
2. Supabase will send a verification email with a link to this URL, plus authentication tokens.
3. If login fails with "email_not_confirmed", we try to automatically confirm the email.
4. The email confirmation can also be triggered manually using the `ConfirmEmail` method.

## Security Considerations

- In production, always use proper email verification flows rather than automatic confirmation.
- Ensure your redirect URL uses HTTPS for secure token transmission.
- The admin API key should be protected and never exposed to clients.

## Testing

To test email confirmation manually:

1. Register a new user
2. Check if an email was sent (in development, check console logs)
3. Click the verification link or use the automatic confirmation
4. Try logging in