// Submits the login/register forms and the logout link to the JSON API with fetch,
// so the browser never navigates to a raw JSON response.

// Shows an error above the form. Reuses the .error box if the server already rendered one,
// otherwise creates it. textContent/append is used instead of innerHTML so the server's
// message can't inject markup.
function showError(form, message) {
  let box = form.parentElement.querySelector('.error');
  if (!box) {
    box = document.createElement('div');
    box.className = 'error';
    form.parentElement.insertBefore(box, form);
  }
  box.textContent = '';
  const label = document.createElement('strong');
  label.textContent = 'Error:';
  box.append(label, ' ', message);
}

// Picks the message to show from an API error body. 422 responses use the
// HTTPValidationError shape ({detail: [{msg}]}), while 401 uses AuthResponse ({message}).
function errorMessage(body, fallback) {
  if (body && body.detail && body.detail[0] && body.detail[0].msg) return body.detail[0].msg;
  if (body && body.message) return body.message;
  return fallback;
}

// Login and register forms: post the fields as form data, then redirect on success
// (both log the user in, so go to the home page) or show the error and stay on the form.
document.querySelectorAll('form[action="/api/login"], form[action="/api/register"]').forEach((form) => {
  form.addEventListener('submit', async (event) => {
    // Stop the normal form submit, which would navigate to the JSON response.
    event.preventDefault();
    try {
      const response = await fetch(form.action, {
        method: 'POST',
        // URLSearchParams makes fetch send application/x-www-form-urlencoded,
        // the same encoding the plain form used.
        body: new URLSearchParams(new FormData(form)),
      });
      // The body may not be JSON (e.g. an empty 500), so don't let parsing throw.
      const body = await response.json().catch(() => null);
      if (response.ok) {
        window.location.href = '/';
      } else {
        showError(form, errorMessage(body, 'Something went wrong'));
      }
    } catch (err) {
      // fetch only rejects on network failure, not on HTTP error statuses.
      showError(form, 'Could not reach the server');
    }
  });
});

// Logout link: /api/logout clears the session cookie and returns JSON, so call it in the
// background and then go to the home page. The redirect runs even if the request fails.
const logout = document.getElementById('nav-logout');
if (logout) {
  logout.addEventListener('click', async (event) => {
    event.preventDefault();
    try {
      await fetch('/api/logout');
    } catch {
      // A network failure is ignored on purpose: the finally block still sends the user
      // home, and catching it here keeps an unhandled rejection out of the console.
    } finally {
      window.location.href = '/';
    }
  });
}
