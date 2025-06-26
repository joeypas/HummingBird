document.getElementById('form').onsubmit = async function(e) {
	e.preventDefault();
	const resp = await fetch('/register', {
		method: 'POST',
		headers: { 'Content-Type': 'application/json' },
		body: JSON.stringify({
			username: document.getElementById('username').value,
			email: document.getElementById('email').value,
			password: document.getElementById('password').value
		})
	});
	if (!resp.ok) {
		alert('registration failed');
		return;
	}
	const data = await resp.json();
	localStorage.setItem('token', data.token);
	location.href = 'index.html';
};
