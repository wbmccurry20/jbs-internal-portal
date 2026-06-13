# Email Automation — Azure AD Configuration

This document describes the one-time Azure portal steps required to enable the JBS portal's
email automation feature. These steps must be completed by a **tenant administrator** before
the feature can be activated.

---

## Background

The JBS portal already has an existing Azure AD app registration used for SharePoint / Microsoft
Graph OAuth2 (delegated authentication). The app credentials (`AZURE_CLIENT_ID`,
`AZURE_CLIENT_SECRET`, `AZURE_TENANT_ID`) are stored in the backend `.env` file.

Email automation requires one additional delegated permission — `Mail.ReadWrite.Shared` — to
allow the portal to read and move messages in the shared JBS Bids mailbox. The existing
`offline_access` and SharePoint scopes are **not affected** by this change.

---

## Prerequisites

- Access to the [Azure portal](https://portal.azure.com) with a **tenant administrator** account
  (admin consent cannot be granted without one)
- The value of `AZURE_CLIENT_ID` from the backend `.env` (used to locate the correct app
  registration)

---

## Steps

### 1. Locate the App Registration

1. In the Azure portal, navigate to **Azure Active Directory → App registrations**.
2. Find the existing JBS portal application. You can search by the value of `AZURE_CLIENT_ID`
   from the backend `.env` (it will be listed under "Application (client) ID").
   - The app display name is referred to below as **[YOUR_APP_NAME]**.

### 2. Add the Mail.ReadWrite.Shared Permission

1. Open the app registration and click **API permissions** in the left sidebar.
2. Click **Add a permission**.
3. Select **Microsoft Graph**.
4. Select **Delegated permissions**.
5. In the search box, type `Mail.ReadWrite.Shared`.
6. Check the box next to **Mail.ReadWrite.Shared** and click **Add permissions**.

> **Why this scope is needed:** `Mail.ReadWrite.Shared` allows the portal to read messages in,
> and move messages out of, the shared JBS Bids mailbox on behalf of the signed-in user. Without
> this permission the email automation feature cannot access the shared mailbox.

### 3. Grant Admin Consent

1. Back on the **API permissions** page, click **Grant admin consent for [YOUR_TENANT_NAME]**.
2. Confirm the prompt when asked.
3. Verify that the `Mail.ReadWrite.Shared` row now shows a green checkmark with status
   **Granted for [YOUR_TENANT_NAME]**.

> **Important:** This step requires a tenant administrator account. If you are not a tenant
> admin, ask your Azure AD administrator to complete this step.

### 4. Add the Redirect URI

1. In the app registration, click **Authentication** in the left sidebar.
2. Under **Redirect URIs**, verify that the value of `MAIL_REDIRECT_URI` (configured as part
   of Ticket 3) is present. If it is not listed, click **Add URI** and paste the value.
3. Click **Save**.

---

## Post-Setup Configuration

Before email automation can run, the shared mailbox address must be saved in the portal's
settings table:

> **Placeholder:** The shared mailbox address must be set in the portal settings page
> (`email_automation_config` table, key: `mailbox_address`) before automation can run.
> Set the value to **[YOUR_MAILBOX_ADDRESS]** (the full SMTP address of the shared JBS Bids
> mailbox, e.g. `bids@example.com`).

---

## Summary of Permission Changes

| Permission              | Type      | Status after setup                          |
|-------------------------|-----------|---------------------------------------------|
| `Mail.ReadWrite.Shared` | Delegated | **New** — admin consent required            |
| `offline_access`        | Delegated | Unchanged                                   |
| SharePoint scopes       | Delegated | Unchanged                                   |

---

## Troubleshooting

| Symptom | Likely cause |
|---------|--------------|
| "Insufficient privileges" error when accessing the mailbox | Admin consent was not granted, or was granted on the wrong app registration |
| Permission shows "Not granted" | Click **Grant admin consent** again as a tenant admin |
| Redirect URI mismatch error during OAuth flow | The `MAIL_REDIRECT_URI` value is missing from the Authentication blade |
