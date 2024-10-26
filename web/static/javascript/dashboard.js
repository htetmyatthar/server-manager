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

	async restartServer(adminUsername, adminPassword, token) {
		try {
			const password = adminPassword.trim();
			const username = adminUsername.trim();
			const response = await fetch('/server', {
				method: 'POST',
				headers: {
					'Content-Type': 'application/json',
					'token': token,
				},
				body: JSON.stringify({ "adminUsername": username, "adminPassword": password }),
			});

			if (!response.ok) {
				if (response.status === 429) {
					alert("Too many request. Try again later.")
				}
				if (response.status === 401) {
					alert('Failed to restart the server. Please check your password.');
				}
				if (response.status === 500) {
					alert('Server Error. Contact the Administrator to fix this.');
				}
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

		// Defined defaults for canvas
		this.fontFamily = "'Roboto', 'Open Sans', 'Noto Serif', sans-serif";
		this.fontSize = '36px';
		this.textColor = '#FF0000';	// RED color
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

	downloadQR() {
		return new Promise((resolve, reject) => {
			const svgElement = this.qrCodeElement.querySelector('svg');
			if (!svgElement) {
				reject(new Error("No QR code found to download."));
				return;
			}

			document.fonts.load(this.fontSize + ' ' + this.fontFamily).then(() => {
				const canvas = document.createElement('canvas');
				const ctx = canvas.getContext('2d');

				// Larger canvas size for better quality
				const canvasWidth = 1024;
				const canvasHeight = 1024;
				const padding = 40; // Padding for border and canvas edge

				canvas.width = canvasWidth;
				canvas.height = canvasHeight;

				const img = new Image();
				const svgData = new XMLSerializer().serializeToString(svgElement);
				const svgBlob = new Blob([svgData], { type: 'image/svg+xml;charset=utf-8' });
				const url = URL.createObjectURL(svgBlob);

				img.onload = () => {
					// Fill white background
					ctx.fillStyle = 'white';
					ctx.fillRect(0, 0, canvasWidth, canvasHeight);

					// Draw border with padding
					ctx.strokeStyle = 'black';
					ctx.lineWidth = 2;
					ctx.strokeRect(padding, padding, canvasWidth - (padding * 2), canvasHeight - (padding * 2));

					// Set up text styling
					ctx.fillStyle = this.textColor;
					ctx.textAlign = 'center';
					ctx.font = `${this.fontSize} ${this.fontFamily}`;

					// Calculate positions for balanced layout
					const qrSize = 750; // Larger QR code
					const qrX = (canvasWidth - qrSize) / 2;
					const contentHeight = qrSize + (this.fontSize.replace('px', '') * 2); // QR height + 2 lines of text
					const topSpace = (canvasHeight - contentHeight) / 2;

					// Draw username at the top with equal spacing
					const username = this.usernameElement.innerText;
					const textY = topSpace + parseInt(this.fontSize); // Position for top text
					ctx.fillText(username, canvasWidth / 2, textY);

					// Draw the QR code
					const qrY = textY + 20; // Small gap after username
					ctx.drawImage(img, qrX, qrY, qrSize, qrSize);

					// Draw remarks at the bottom with equal spacing
					const remarks = this.remarksElement.innerText;
					const bottomTextY = qrY + qrSize + parseInt(this.fontSize); // Position for bottom text
					ctx.fillText(remarks, canvasWidth / 2, bottomTextY);

					// Convert to blob and download
					canvas.toBlob((blob) => {
						const downloadUrl = URL.createObjectURL(blob);
						const downloadLink = document.createElement('a');
						downloadLink.href = downloadUrl;
						downloadLink.download = `qr-code-${username || 'user'}.png`;
						document.body.appendChild(downloadLink);
						downloadLink.click();
						document.body.removeChild(downloadLink);
						URL.revokeObjectURL(downloadUrl);
						URL.revokeObjectURL(url);
						resolve();
					}, 'image/png');
				};

				img.onerror = () => {
					URL.revokeObjectURL(url);
					reject(new Error("Failed to load QR code image."));
				};

				img.src = url;
			});
		});
	}
}

document.addEventListener("DOMContentLoaded", () => {
	const reStartModal = new Modal("#modal", "#serverRestartModalBtn", ".closeModalBtn", "#serverRestartBtn");
	new Modal("#qrModal", ".qrBtn", ".closeModalBtn", ".closeModalBtn");
	const server = new Server();
	const qrGenerator = new QRGenerator();

	const today = new Date(); // Get the current date
	today.setDate(today.getDate() + 1); // Add one day

	// Set the input's valueAsDate to the new date (one day ahead)
	const dateInput = document.getElementById("formStartDate");
	dateInput.valueAsDate = today;

	document.getElementById("uuidManualButton").addEventListener("click", () => {
		document.querySelector("#serverUUID").value = "";
		document.querySelector("#deviceUUID").value = "";
	});

	document.getElementById("usernameInput").addEventListener("input", () => {
		if (usernameInput.value.trim() !== "") {
			document.querySelector("#serverUUID").value = generateUUID();
		}
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
		const adminUsername = document.getElementById("adminUsername").value;
		if (adminUsername === "") {
			alert("Username required!")
			reStartModal.close();
			return
		}
		const token = document.getElementById("CSRFToken").value;
		await server.restartServer(adminUsername, adminPassword, token);
		reStartModal.close();
	});

	document.getElementById("closeQrModalBtn").addEventListener("click", async () => {
		try {
			if (document.getElementById("qrCode").innerHTML) {
				await qrGenerator.downloadQR();  // Wait for download to complete
				qrGenerator.cleanQR();  // Clean up only after successful download
			}
		} catch (error) {
			console.error("Failed to download QR code:", error);
			alert("Failed to download QR code. Please try again.");
		}
	});

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
	const address = window.location.hostname; // Hardcoded for simplicity, consider making it configurable
	const subDomain = address.split(".")[0];
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
		scy: "none",
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
	const address = window.location.hostname; // Hardcoded for simplicity, consider making it configurable
	const subDomain = address.split(".")[0];
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
		scy: "none",
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
