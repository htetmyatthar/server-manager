class QRInfo {
	constructor(userNumber, username, deviceID, id, expDate) {
		this.userNumber = userNumber;
		this.username = username;
		this.expDate = expDate;
		this.id = id;
		this.deviceID = deviceID;
	}
}

class Modal {
	constructor(modalSelector, openBtnSelector, closeBtnSelector) {
		this.modal = document.querySelector(modalSelector);
		this.openBtns = document.querySelectorAll(openBtnSelector);
		this.closeBtns = document.querySelectorAll(closeBtnSelector);
		this.init();
	}

	init() {
		this.openBtns.forEach((openBtn) => {
			openBtn.addEventListener("click", () => this.open());
		});
		this.closeBtns.forEach((closeBtn) => {
			closeBtn.addEventListener("click", () => this.close())
		});
	}

	open() {
		this.modal.showModal();
	}

	close() {
		this.modal.close();
	}
}

class Server {
	constructor() {
		this.serverIPDisplay = document.querySelector("#serverIP");
		this.port = document.getElementById("v2rayServerPort").dataset.port;
	}

	async refreshIP() {
		try {
			const response = await fetch("/server/ip");
			if (!response.ok) throw new Error("Network response was not ok");
			const data = await response.json();
			this.serverIPDisplay.innerText = data.ip || "No IP found.";
			alert("IP refreshed.");
		} catch (error) {
			console.error("Error fetching IP: ", error);
			this.serverIPDisplay.innerText = "Failed to get IP address.";
		}
	}

	async restartServer(adminPassword, token) {
		try {
			const password = adminPassword.trim();
			const response = await fetch('/server', {
				method: 'POST',
				headers: {
					'Content-Type': 'application/json',
					'token': token,
				},
				body: JSON.stringify({ "adminPassword": password }),
			});

			if (!response.ok) {
				alert('Failed to restart the server. Please check your password.');
			} else {
				alert('Server is restarting...');
			}
		} catch (error) {
			console.error('Error:', error);
			alert('An error occurred. Please try again.');
		}
	}
}

class QRGenerator {
	constructor() {
		this.qrCodeElement = document.getElementById('qrCode');
		this.usernameElement = document.getElementById('qrUsername');
		this.remarksElement = document.getElementById('qrRemarks');
	}

	async cleanQR() {
		this.qrCodeElement.innerHTML = "";
		this.usernameElement.innerHTML = "";
		this.remarksElement.innerHTML = "";
	}

	async generateQR(qrData, generateURIFunc, isLocked) {
		if (!qrData) {
			console.error("There's no QR data to be encoded.");
			return;
		}
		const encodedData = generateURIFunc(qrData);
		try {
			const svg = await QRCode.toString(encodedData, { type: 'svg' });
			this.qrCodeElement.innerHTML = svg;

			if (isLocked) {
				this.usernameElement.innerText = `${qrData.username} ${qrData.deviceID.slice(-4)}`;
			} else {
				this.usernameElement.innerText = qrData.username;
			}

			const subDomain = window.location.hostname.split(".")[0];
			this.remarksElement.innerText = `${subDomain} - ${qrData.id.slice(-4)}`;
		} catch (error) {
			console.error(error);
			alert("Failed to generate QR code.");
		}
	}
}

document.addEventListener("DOMContentLoaded", () => {
	const reStartModal = new Modal("#modal", "#serverRestartModalBtn", ".closeModalBtn", "#serverRestartBtn");
	new Modal("#qrModal", ".qrBtn", ".closeModalBtn", ".closeModalBtn");
	const server = new Server();
	const qrGenerator = new QRGenerator();

	document.getElementById("uuidGenerateButton").addEventListener("click", () => {
		document.querySelector("#serverUUID").value = generateUUID();
	});

	document.getElementById("serverIPRefreshBtn").addEventListener("click", () => server.refreshIP());

	document.getElementById("userRefreshBtn").addEventListener("click", () => window.location.reload());

	document.getElementById("serverRestartBtn").addEventListener('click', async (event) => {
		event.preventDefault();
		const adminPassword = document.getElementById("adminPassword").value;
		if (adminPassword === "") {
			alert("Password required!")
			reStartModal.close();
			return
		}
		const token = document.getElementById("CSRFToken").value;
		await server.restartServer(adminPassword, token);
		reStartModal.close();
	});

	document.getElementById("closeQrModalBtn").addEventListener("click", () => {
		qrGenerator.cleanQR();
	})

	document.querySelectorAll(".copyBtn").forEach((copyBtn) => {
		copyBtn.addEventListener("click", () => {
			const row = copyBtn.closest("tr");
			const qrData = new QRInfo(
				row.cells[0].dataset.value,
				row.cells[1].innerText,
				row.cells[2].innerText,
				row.cells[3].innerText,
				row.cells[5].innerText,
			);
			navigator.clipboard.writeText(generateURI(qrData))
				.then(() => alert("Copied the text: " + generateURI(qrData)))
				.catch(err => {
					console.error('Failed to copy: ', err);
					alert("Failed to copy the text.");
				});
		});
	});

	const handleQRButtonSubmit = async (event) => {
		event.preventDefault();

		const form = event.target;
		let userNumber = form.elements["qrUserNumber"].value;

		if (userNumber === "") {
			alert("Please enter user number.");
			return;
		}

		// Convert userNumber to string or integer based on how it's stored in data-value
		userNumber = userNumber.trim(); // If data-value is string, trim any extra spaces

		let generateURIFunc;
		let isLocked = true;
		if (form.classList.contains("open")) {
			isLocked = false;
			generateURIFunc = generateURI;
		} else {
			generateURIFunc = generateLockedURI;
		}

		// Convert data-value and userNumber to the same type
		const userRow = Array.from(document.querySelectorAll('.user-table tbody tr')).find(row => {
			const rowNumber = row.cells[0].dataset.value.trim(); // Make sure there are no spaces
			return rowNumber === userNumber; // Compare as strings
		});

		if (userRow) {
			const qrData = new QRInfo(
				userRow.cells[0].dataset.value,
				userRow.cells[1].innerText,
				userRow.cells[2].innerText,
				userRow.cells[3].innerText,
				userRow.cells[5].innerText,
			);
			await qrGenerator.generateQR(qrData, generateURIFunc, isLocked);
		} else {
			alert("User number not found.");
		}
	};

	// Attach event listeners
	document.querySelectorAll(".qrForm").forEach((form) => {
		form.addEventListener("submit", handleQRButtonSubmit);
	});
});

function generateURI(qrData) {
	const prefix = "vmess://";
	const address = "sg1-v2.lothone.shop"; // Hardcoded for simplicity, consider making it configurable
	const subDomain = window.location.hostname.split(".")[0];
	const port = document.getElementById("v2rayServerPort").dataset.port;

	const vmessTemplate = {
		add: address,
		aid: "1",
		alpn: "",
		fp: "",
		host: "www.youtube.com",
		id: qrData.id,
		net: "tcp",
		path: "/",
		port: port,
		ps: `valid before (${qrData.expDate}) ${subDomain}-Singapore-${qrData.id.slice(-4)}`,
		scy: "aes-128-gcm",
		sni: "",
		tls: "",
		type: "http",
		v: "2"
	};

	return `${prefix}${btoa(JSON.stringify(vmessTemplate))}`;
}

function generateLockedURI(qrData) {
	if (qrData.deviceID === "") {
		alert("Unable to generate locked QR without device id.")
		return
	}
	const prefix = "vmess://";
	const lockPrefix = "v2box://locked=";
	const address = "sg1-v2.lothone.shop"; // Hardcoded for simplicity, consider making it configurable
	const subDomain = window.location.hostname.split(".")[0];
	const port = document.getElementById("v2rayServerPort").dataset.port;

	const vmessTemplate = {
		add: address,
		aid: "1",
		alpn: "",
		fp: "",
		host: "www.youtube.com",
		deviceID: qrData.deviceID,
		id: qrData.id,
		net: "tcp",
		path: "/",
		port: port,
		ps: `valid before (${qrData.expDate}) ${subDomain}-Singapore-${qrData.id.slice(-4)}`,
		scy: "aes-128-gcm",
		sni: "",
		tls: "",
		type: "http",
		v: "2"
	};

	const unlockedQR = `${prefix}${btoa(JSON.stringify(vmessTemplate))}`;
	return `${lockPrefix}${btoa(unlockedQR)}`;
}

function generateUUID() {
	return 'xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx'.replace(/[xy]/g, function(c) {
		const r = Math.random() * 16 | 0;
		return c === 'x' ? r.toString(16) : (r & 0x3 | 0x8).toString(16);
	});
}
