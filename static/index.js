function go(id, name) { location.href = `chat.html?room=${id}&name=${name}`; }
function logout() { localStorage.removeItem('token'); location.href = 'login.html'; }
if (!localStorage.getItem('token')) {
	location.href = 'login.html';
}

(async function loadRooms() {
	try {
		const resp = await fetch('/rooms', {
			method: 'GET',
		});
		if (!resp.ok) throw new Error(await resp.text());

		const rooms = await resp.json();           // → ["general", "music", ...]
		const container = document.getElementById('rooms');

		rooms.rooms.forEach(room => {
			const btn = document.createElement('button');
			btn.textContent = room.name.charAt(0).toUpperCase() + room.name.slice(1);
			btn.onclick = () => go(room.id, room.name);
			container.appendChild(btn);
		});
	} catch (err) {
		console.error('Room fetch failed:', err);
		alert('Could not load rooms. Please try again later.');
	}
})();
