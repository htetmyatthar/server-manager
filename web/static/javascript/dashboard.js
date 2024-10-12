class QRInfo {
	constructor(userNumber, username, expDate, id, deviceID) {
		this.userNumber = userNumber;
		this.username = username;
		this.expDate = expDate;
		this.id = id;
		this.deviceID = deviceID
	}
}

document.addEventListener("DOMContentLoaded", () => {
	const modal = document.querySelector("#modal");
	const openModal = document.querySelector(".openBtn");
	const closeModal = document.querySelector(".closeBtn");
	const uuidButton = document.querySelector("#uuidGenerateButton");
	const serverUUID = document.querySelector("#serverUUID");
	const qrBtn = document.querySelector("#qrBtn");
	const lockedQrBtn = document.querySelector("#lockedQrBtn");
	const copyBtns = document.querySelectorAll(".copyBtn");
	const serverIPBtn = document.querySelector("#serverIPRefreshBtn");
	const serverRestartForm = document.querySelector("#restartForm");

	serverRestartForm.addEventListener('submit', async (event) => {
		event.preventDefault(); // Prevent the form from reloading the page
		const adminPassword = document.getElementById("adminPassword").value;
		const token = document.getElementById("CSRFToken").value;
		console.log(adminPassword);
		try {
			const response = await fetch('/server', {
				method: 'POST',
				headers: {
					'Content-Type': 'application/json',
					'token': token,
				},
				body: JSON.stringify({ "adminPassword": adminPassword }),  // Send password in JSON format
			});

			if (response.ok) {
				// Server restart was successful, handle success here
				alert('Server is restarting...');
				modal.close();
			} else {
				console.log(response.status)
				// Server restart failed, handle error here
				alert('Failed to restart the server. Please check your password.');
			}
		} catch (error) {
			console.error('Error:', error);
			alert('An error occurred. Please try again.');
		}
	});

	serverIPBtn.addEventListener("click", () => {
		const ipDisplay = document.querySelector("#serverIP");
		fetch("/server/ip")
			.then(response => {
				if (!response.ok) {
					throw new Error("Network response was not ok");
				}
				return response.json();
			})
			.then(data => {
				console.log(data)
				if (data) {
					ipDisplay.innerText = data.ip;
					alert("ip refreshed.")
				}
			})
			.catch(error => {
				console.error("Error fetching IP: ", error);
				ipDisplay.innerText = "Failed to get IP address.";
			})
	})

	copyBtns.forEach((copyBtn) => {
		copyBtn.addEventListener("click", () => {
			const row = copyBtn.closest("tr");
			const qrData = new QRInfo(
				row.cells[1].innerText,
				row.cells[2].innerText,
				row.cells[6].innerText,
				row.cells[4].innerText,
				row.cells[3].innerText,
			);
			const configURI = generateURI(qrData);
			navigator.clipboard.writeText(configURI)
				.then(() => {
					alert("Copied the text: " + configURI);
				})
				.catch((err) => {
					console.error('Failed to copy: ', err);
					alert("Failed to copy the text.");
				});
		});
	});

	qrBtn.addEventListener("click", () => {
		const userNumber = document.querySelector("input[name='qrUserNumber']").value;
		if (!userNumber) {
			alert("Please enter user number.");
			return;
		}

		const userRow = Array.from(document.querySelectorAll('.user-table tbody tr')).find(row => {
			return row.cells[1].innerText === userNumber;
		});

		if (userRow) {
			const qrData = new QRInfo(
				userRow.cells[1].innerText,
				userRow.cells[2].innerText,
				userRow.cells[6].innerText,
				userRow.cells[4].innerText,
				userRow.cells[3].innerText,
			);
			// Generate the QR code using the qrData
			generateQR(qrData, generateURI);
		} else {
			alert("User number not found.");
		}
	});

	lockedQrBtn.addEventListener("click", () => {
		const userNumber = document.querySelector("input[name='lockedQrUserNumber']").value;
		if (!userNumber) {
			alert("Please enter user number.");
			return;
		}

		const userRow = Array.from(document.querySelectorAll('.user-table tbody tr')).find(row => {
			return row.cells[1].innerText === userNumber;
		});

		if (userRow) {
			const qrData = new QRInfo(
				userRow.cells[1].innerText,
				userRow.cells[2].innerText,
				userRow.cells[6].innerText,
				userRow.cells[4].innerText,
				userRow.cells[3].innerText,
			);
			// Generate the QR code using the qrData
			generateQR(qrData, generateLockedURI);
		} else {
			alert("User number not found.");
		}
	});

	uuidButton.addEventListener("click", () => {
		// serverUUID.value = crypto.randomUUID();
		serverUUID.value = generateUUID();
	});

	openModal.addEventListener("click", () => {
		modal.showModal();
	});

	closeModal.addEventListener("click", () => {
		modal.close();
	});
});

function generateURI(qrData) {
	const prefix = "vmess://";
	const address = window.location.hostname;
	const subDomain = address.split(".")[0];
	const port = document.getElementById("v2rayServerPort").dataset.port;

	// Note: field names should be lowercase as per VMess protocol
	const vmessTemplate = {
		// add: address,
		add: "sg1-v2.lothone.shop",
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
	const jsonConfig = JSON.stringify(vmessTemplate);
	console.log(jsonConfig)

	// Base64 encode the JSON string
	const base64Config = btoa(jsonConfig);
	return `${prefix}${base64Config}`;
}

// compatable with v2box locked and share feature.
function generateLockedURI(qrData) {
	const prefix = "vmess://";
	const lockPrefix = "v2box://locked="
	const address = window.location.hostname;
	const subDomain = address.split(".")[0];
	const port = document.getElementById("v2rayServerPort").dataset.port;

	// Note: field names should be lowercase as per VMess protocol
	const vmessTemplate = {
		// add: address,
		add: "sg1-v2.lothone.shop",
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
	const jsonConfig = JSON.stringify(vmessTemplate);
	console.log(jsonConfig)

	// Base64 encode the JSON string
	const base64Config = btoa(jsonConfig);
	const unlockedQR = `${prefix}${base64Config}`;

	// Base64 encode the unlockedQR.
	const lockedConfig = btoa(unlockedQR);
	return `${lockPrefix}${lockedConfig}`
}

// dependency injection
function generateQR(qrData, generateURIFunc) {
	if (!qrData) {
		console.error("There's no qr data to be encoded.")
		return
	}
	const encodedData = generateURIFunc(qrData)
	const qrCodeContainer = document.getElementById('qrCode');

	// clean previous qr if there's any.
	qrCodeContainer.innerHTML = "";

	// Generate QR code as SVG
	QRCode.toString(encodedData, { type: 'svg', width: 225 }, function(error, svg) {
		if (error) {
			console.error(error);
			alert("Failed to generate QR code.");
			return;
		}
		qrCodeContainer.innerHTML = svg;
	});

	// add the username and remarks for the qr.
	if (qrData.deviceID === "") {
		document.getElementById('qrUsername').innerText = `${qrData.username}`;
	} else {
		document.getElementById('qrUsername').innerText = `${qrData.username}${qrData.deviceID.slice(-4)}`;
	}
	const subDomain = window.location.hostname.split(".")[0];
	document.getElementById('qrRemarks').innerText = `${subDomain} - ${qrData.id.slice(-4)}`;
}

function generateUUID() {
	return 'xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx'.replace(/[xy]/g, function(c) {
		const r = Math.random() * 16 | 0;
		const v = c === 'x' ? r : (r & 0x3 | 0x8);
		return v.toString(16);
	});
}
