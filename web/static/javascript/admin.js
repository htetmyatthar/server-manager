document.addEventListener("DOMContentLoaded", () => {
	const loginBtn = document.querySelector
	const inputFields = document.querySelectorAll(".inputField")
	inputFields.forEach((inputField) => {
		disableBtn(inputField, loginBtn);
	})
});

/**
 * disableBtn disable a button until the required parameter is filled in.
 *
 * @param {HTMLButtonElement | HTMLInputElement} btn - button that is going to be disabled.
 * @param {HTMLInputElement} inputField - inputField that will enble the button.
 *
 * @example
 * const input = document.querySelector("input");
 * const button = document.querySelector("button");
 * disableBtn(input, button);	// Output: the button will be enable when the input is not empty.
*/
function disableBtn(inputField, btn) {
    btn.disabled = true;
    inputField.addEventListener("input", function() {
        if (inputField.value.trim() !== "") {
            btn.disabled = false;
        } else {
            btn.disabled = true;
        }
    });
}
