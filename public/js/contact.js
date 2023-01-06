// For Contact Form
function submitData() {
	let name = document.getElementById("name").value;
	let email = document.getElementById("email").value;
	let phone_number = document.getElementById("phone_number").value;
	let subject = document.getElementById("subject").value;
	let message = document.getElementById("message").value;

	// console.log(name, email, phone_number, subject, message);

	if (name === "") {
		return alert("Name not found")
	} else if (email == "") {
		return alert("Email not found")
	} else if (phone_number == "") {
		return alert("Phone Number not found")
	} else if (subject == "" ) {
		return alert("Subject not found")
	} else if (message == "") {
		return alert("Massage not found")
	}

	let link = document.createElement("a");
	link.href = `mailto:${email}?subject=${subject}&body=Halo nama saya ${name}, pesan saya ${message}, silahkan kontak nomor saya di ${phone_number}. Terimakasih.`;
	link.click();
}