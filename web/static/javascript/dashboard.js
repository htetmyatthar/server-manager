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
	// Close all action menus when clicking outside
	document.addEventListener('click', function(e) {
		if (!e.target.closest('.actions-container')) {
			const allActions = document.querySelectorAll('.actions');
			const allButtons = document.querySelectorAll('.show-actions-btn');
			allActions.forEach(actions => actions.classList.remove('show'));
			allButtons.forEach(button => button.classList.remove('active'));
		}
	});

	// Toggle action menu
	const showActionButtons = document.querySelectorAll('.show-actions-btn');
	showActionButtons.forEach(button => {
		button.addEventListener('click', function(e) {
			e.stopPropagation();

			// Close all other action menus
			const allActions = document.querySelectorAll('.actions');
			const allButtons = document.querySelectorAll('.show-actions-btn');
			allActions.forEach(actions => {
				if (actions !== this.nextElementSibling) {
					actions.classList.remove('show');
				}
			});
			allButtons.forEach(btn => {
				if (btn !== this) {
					btn.classList.remove('active');
				}
			});

			// Toggle current action menu
			const actionsMenu = this.nextElementSibling;
			actionsMenu.classList.toggle('show');
			this.classList.toggle('active');
		});
	});

	// restart server modal
	const reStartModal = new Modal("#modal", "#serverRestartModalBtn", ".closeModalBtn", "#serverRestartBtn");

	// user edit form modal
	const userUpdateModal = new Modal("#userUpdateModal", "#qrUserNumber", ".closeModalBtn");

	// qr display modal
	const qrModal = new Modal("#qrModal", ".qrBtn", ".closeModalBtn", ".closeModalBtn");

	// delete user modal
	const deleteUserModal = new Modal("#deleteUserModal", "#deleteUserModalBtn", ".closeModalBtn");

	// qr type choose modal
	const generateQRUserModal = new Modal("#generateUserQRModal", "#qrUserNumber", ".closeModalBtn");

	const server = new Server();
	const qrGenerator = new QRGenerator();
	const today = new Date();
	today.setDate(today.getDate());

	// default value start date.
	const dateInput = document.getElementById("formStartDate");
	dateInput.valueAsDate = today;

	// qr code generations logic
	const handleQRButtonClick = async (event) => {
		const button = event.target;
		const userNumber = document.getElementById("qrUserNumber").dataset.value;
		if (userNumber === "") {
			console.log("Error trying to find the qrUserNumber.")
			return;
		}

		let generateURIFunc;
		let isLocked = true;
		if (button.classList.contains("open")) {
			isLocked = false;
			generateURIFunc = generateURI;
		} else {
			generateURIFunc = generateLockedURI;
		}

		// convert data-value and userNumber to the same type
		const userRow = Array.from(document.querySelectorAll('.user-table tbody tr')).find(row => {
			const rowNumber = row.cells[0].dataset.value.trim(); // Make sure there are no spaces
			return rowNumber === userNumber; // Compare as strings
		});

		// the user row will be definitely found.
		const qrData = new QRInfo(
			userRow.cells[0].dataset.value,
			userRow.cells[1].innerText,
			userRow.cells[2].innerText,
			userRow.cells[3].innerText,
			userRow.cells[5].innerText,
		);
		await qrGenerator.generateQR(qrData, generateURIFunc, isLocked);
		qrModal.open();
	};

	// opened qr code generation handler
	document.querySelector("#openQRBtn").addEventListener("click", (event) => {
		generateQRUserModal.close();
		handleQRButtonClick(event);
	})

	// locked qr code generation handler
	document.querySelector("#deviceLockedQRBtn").addEventListener("click", (event) => {
		generateQRUserModal.close();
		handleQRButtonClick(event);
	})

	// user edit buttons handler
	document.querySelectorAll(".editBtn").forEach((button) => {
		button.addEventListener("click", () => {
			// alert("Feature is not supported yet.")
			const row = button.closest("tr");
			const numberCell = row.querySelector("[data-cell='Number']");
			const startDateCell = row.querySelector("[data-cell='Start date']")
			const expireDateCell = row.querySelector("[data-cell='Expire date']")
			const serverUUIDCell = row.querySelector("[data-cell='Server UUID']");
			const deviceUUIDCell = row.querySelector("[data-cell='Device UUID']");
			console.log(deviceUUIDCell);
			const usernameCell = row.querySelector("[data-cell='Username']");

			const userNumber = numberCell ? numberCell.dataset.value : "not found";
			const startDate = startDateCell ? startDateCell.dataset.value : "not found";
			const expireDate = expireDateCell ? expireDateCell.dataset.value : "not found";
			const serverUUID = serverUUIDCell ? serverUUIDCell.dataset.value : "not found";
			const deviceUUID = deviceUUIDCell ? deviceUUIDCell.dataset.value : "not found";
			const username = usernameCell ? usernameCell.dataset.value : "not found";

			// userNumber
			document.querySelector("#userUpdateUserNumber").value = userNumber;

			// username
			const usernameInput = document.querySelector("#userUpdateUsername");
			usernameInput.value = username;

			// server uuid
			const serverUUIDInput = document.querySelector("#userUpdateServerId")
			serverUUIDInput.value = serverUUID;

			// device uuid
			const deviceUUIDInput = document.querySelector("#userUpdateDeviceId")
			console.log(deviceUUID);
			deviceUUIDInput.value = deviceUUID;

			// start date
			const startDateInput = document.querySelector("#userUpdateStartDate")
			startDateInput.value = startDate;
			console.log(startDateInput, startDate)

			// start date
			const expireDateInput = document.querySelector("#userUpdateExpireDate")
			expireDateInput.value = expireDate;
			console.log(expireDateInput, expireDate)

			// open the user update modal
			userUpdateModal.open();
		})
	})

	// generate qr buttons handler
	document.querySelectorAll(".generateQRBtn").forEach((button) => {
		button.addEventListener("click", () => {
			const row = button.closest("tr");
			const numberCell = row.querySelector("[data-cell='Number']");
			const userNumber = numberCell ? numberCell.dataset.value : "not found";
			document.querySelector("#qrUserNumber").dataset.value = userNumber;
			// open the modal to start deleting the user.
			generateQRUserModal.open();
		});
	});

	// user delete buttons handler
	document.querySelectorAll(".deleteBtn").forEach((button) => {
		button.addEventListener("click", () => {
			const row = button.closest("tr");
			const numberCell = row.querySelector("[data-cell='Number']");
			const UUIDCell = row.querySelector("[data-cell='Server UUID']");
			const usernameCell = row.querySelector("[data-cell='Username']");

			const userNumber = numberCell ? numberCell.dataset.value : "not found";
			const serverUUID = UUIDCell ? UUIDCell.dataset.value : "not found";
			const username = usernameCell ? usernameCell.dataset.value : "not found";

			document.querySelector("#userToBeDeleted").innerHTML = username;	// user
			document.querySelector("#userNumberToBeDeleted").value = userNumber;	// usernumber

			// uuid
			const serverUUIDInput = document.querySelector("#serverUUIDToBeDeleted")
			serverUUIDInput.dataset.check = serverUUID;
			serverUUIDInput.placeholder = serverUUID.slice(-4);

			// username
			const usernameInput = document.querySelector("#usernameToBeDeleted");
			usernameInput.dataset.check = username;
			usernameInput.placeholder = username;
			// open the delete user modal
			deleteUserModal.open();
		})
	})


	// user delete confirm form handler
	document.getElementById("deleteUserModalBtn").addEventListener('click', async () => {
		const form = document.querySelector("#userDeleteForm");
		const usernameInput = document.getElementById("usernameToBeDeleted");
		const serverUUIDInput = document.getElementById("serverUUIDToBeDeleted");

		// check for empty inputs. 
		if (serverUUIDInput.value === "" || usernameInput.value === "") {
			alert("You type in the wrong last 4 digits of server UUID.")
			deleteUserModal.close();
			return
		}

		// server id input check and replace if correct.
		if (serverUUIDInput.value !== serverUUIDInput.dataset.check.slice(-4)) {
			alert("You type in the wrong last 4 digits of server UUID.")
			deleteUserModal.close();
			return
		} else {
			serverUUIDInput.value = serverUUIDInput.dataset.check;
		}

		// username input check
		if (usernameInput.value !== usernameInput.dataset.check) {
			alert("You type in the wrong username.")
			deleteUserModal.close();
			return
		}
		form.submit();
		deleteUserModal.close();
	});


	// search box handler
	document.getElementById("searchInput").addEventListener("keyup", function() {
		const filter = this.value.toLowerCase();
		const rows = document.querySelectorAll("#userTable tbody tr");

		rows.forEach(row => {
			const cells = row.querySelectorAll("td");
			const match = Array.from(cells).slice(0, -1).some(cell =>
				cell.textContent.toLowerCase().includes(filter)
			);
			row.style.display = match ? "" : "none";
		});
	});

	// manually adding uuid button handler
	document.getElementById("uuidManualButton").addEventListener("click", () => {
		document.querySelector("#serverUUID").value = "";
		document.querySelector("#deviceUUID").value = "";
	});

	// auto generate server uuid when username is inputted.
	document.getElementById("usernameInput").addEventListener("input", () => {
		if (usernameInput.value.trim() !== "") {
			document.querySelector("#serverUUID").value = generateUUID();
		}
	});

	// server ip refresh button handler.
	document.getElementById("serverIPRefreshBtn").addEventListener("click", () => server.refreshIP());

	// user list refresh button handler.
	document.getElementById("userRefreshBtn").addEventListener("click", () => window.location.reload());

	// server restart button handler
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

	// download qr button handler
	document.getElementById("downloadQRBtn").addEventListener("click", async () => {
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

	// close qr modal button handler
	document.getElementById("closeQrModalBtn").addEventListener("click", async () => {
		if (document.getElementById("qrCode").innerHTML) {
			qrGenerator.cleanQR();  // Clean up only after done
		}
	})

	// copy config button handler
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


});

function generateURI(qrData) {
	const prefix = "vmess://";
	const address = window.location.hostname;
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
