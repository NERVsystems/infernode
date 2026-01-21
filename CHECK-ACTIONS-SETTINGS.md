# Steps to Fix GitHub Actions

**You need to check these settings - I don't have permission to access them**

## Step 1: Check Repository Actions Settings

Go to: https://github.com/NERVsystems/infernode/settings/actions

Look for:
1. **Actions permissions** section
   - Should be: "Allow all actions and reusable workflows"
   - If disabled or restricted, enable it

2. **Workflow permissions**
   - Should be: "Read and write permissions"
   - Enable "Allow GitHub Actions to create and approve pull requests"

3. **Fork pull request workflows**
   - Can leave default

## Step 2: Check Organization Settings

Go to: https://github.com/organizations/NERVsystems/settings/actions

Check:
1. **Policies**
   - "Allow all actions and reusable workflows" should be enabled

2. **Repository access**
   - infernode should be in allowed list

3. **Quota**
   - Check if Actions minutes exhausted

## Step 3: Check Billing

Go to: https://github.com/organizations/NERVsystems/settings/billing

Check:
1. **GitHub Actions**
   - Minutes used vs available
   - Payment method valid
   - No outstanding bills

## What I Found

**Symptom:** All workflows fail in 3-4 seconds with 0 steps executed

**What this means:** GitHub Actions can't even START jobs

**Test:** Even `echo "hello"` fails instantly

**Conclusion:** Actions is blocked/disabled at repository or organization level

## Quick Test

After checking settings, push any commit:
```bash
git commit --allow-empty -m "Test Actions"
git push
```

Then check:
```bash
gh run list --repo NERVsystems/infernode --limit 1
```

If it runs longer than 10 seconds, Actions is working again.

---

**I cannot fix this from code. This requires your repository admin access to check/change settings.**
