// @ts-check
const { test, expect } = require('@playwright/test');

// The simulator's browser users (phase 2) click these selectors, so this test walks through
// every one of them like a user would: search, register, log in, log out.
test('search, register, log in and log out through the required selectors', async ({ page }) => {
  // Unique per run, so the test also works against a database that already has users
  const username = `e2e-${Date.now()}`;
  const password = 'e2e-password';

  await test.step('front page shows the search field and the login link', async () => {
    await page.goto('/');
    await expect(page.locator('#search-input')).toBeVisible();
    await expect(page.locator('#search-button')).toBeVisible();
    await expect(page.locator('#nav-login')).toBeVisible();
  });

  await test.step('searching puts the query in the URL', async () => {
    await page.locator('#search-input').fill('test');
    await page.locator('#search-button').click();
    await expect(page).toHaveURL(/[?&]q=test/);
    await expect(page.locator('#search-input')).toHaveValue('test');
  });

  await test.step('register', async () => {
    await page.goto('/register');
    await page.locator('#username').fill(username);
    await page.locator('#email').fill(`${username}@example.com`);
    await page.locator('#password').fill(password);
    await page.locator('#password2').fill(password);
    await page.locator('input[type="submit"]').click();
    // The form posts straight to /api/register, so the browser shows its JSON answer
    await expect(page.locator('body')).toContainText('User registered successfully');
  });

  await test.step('log in', async () => {
    await page.goto('/');
    await page.locator('#nav-login').click();
    await page.locator('input[name="username"]').fill(username);
    await page.locator('input[name="password"]').fill(password);
    await page.locator('#login-button').click();
    await expect(page.locator('body')).toContainText('Logged in successfully');

    await page.goto('/');
    await expect(page.locator('#nav-logout')).toBeVisible();
    await expect(page.locator('#nav-login')).toHaveCount(0);
  });

  await test.step('log out', async () => {
    await page.locator('#nav-logout').click();
    await expect(page.locator('body')).toContainText('Logged out successfully');

    await page.goto('/');
    await expect(page.locator('#nav-login')).toBeVisible();
    await expect(page.locator('#nav-logout')).toHaveCount(0);
  });
});
