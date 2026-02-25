const api = import.meta.env.VITE_AUTH_URL || '';

/**
 * @typedef {Object} LocalAuthResponse
 * @property {string} access_token
 */

/**
 * Performs local login.
 *
 * @param {string} username
 * @param {string} password
 * @return {Promise<Response>} A promise that resolves to the fetch response.
 */
export const localLogin = (username, password) => {
  const formData = new FormData();
  formData.set('grant_type', 'password');
  formData.set('username', username);
  formData.set('password', password);
  return fetch(`${api}/oauth2/token`, {
    method: 'POST',
    body: formData
  });
};
