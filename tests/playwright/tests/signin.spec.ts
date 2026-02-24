import { test, expect } from '@playwright/test';

test('sign in with Google', async ({ page }) => {
  await page.goto('http://localhost:8080/');
  await expect(page.getByRole('button', { name: 'Sign in with Google' })).toBeVisible();
  
  const signInPagePromise = page.waitForEvent('popup');
  await page.getByRole('button', { name: 'Sign in with Google' }).click();
  const signInPage = await signInPagePromise;
  await signInPage.getByRole('button', { name: 'Add new account' }).click();
  await signInPage.getByRole('button', { name: 'Auto-generate user information' }).click();
  await signInPage.getByRole('button', { name: 'Sign in with Google.com' }).click();
  
  await expect(page.getByRole('button', { name: 'Sign out' })).toBeVisible();
  await page.getByRole('button', { name: 'Sign out' }).click();
  await expect(page.getByRole('button', { name: 'Sign in with Google' })).toBeVisible();
});