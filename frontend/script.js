// Email validation regex
const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;

// Get DOM elements
const form = document.getElementById('inviteForm');
const submitBtn = document.getElementById('submitBtn');
const statusMessage = document.getElementById('statusMessage');

// Validation function
function validate() {
    let isValid = true;

    // Clear previous errors
    clearErrors();

    // Validate sender email
    const senderEmail = document.getElementById('senderEmail').value.trim();
    if (!senderEmail) {
        showFieldError('senderEmailError', 'Your email is required');
        isValid = false;
    } else if (!emailRegex.test(senderEmail)) {
        showFieldError('senderEmailError', 'Please enter a valid email address');
        isValid = false;
    }

    // Validate recipient email
    const recipientEmail = document.getElementById('recipientEmail').value.trim();
    if (!recipientEmail) {
        showFieldError('recipientEmailError', 'Recipient email is required');
        isValid = false;
    } else if (!emailRegex.test(recipientEmail)) {
        showFieldError('recipientEmailError', 'Please enter a valid email address');
        isValid = false;
    }

    // Validate datetime
    const datetime = document.getElementById('datetime').value;
    if (!datetime) {
        showFieldError('datetimeError', 'Meeting date and time is required');
        isValid = false;
    }

    return isValid;
}

// Helper function to show field errors
function showFieldError(errorId, message) {
    const errorElement = document.getElementById(errorId);
    errorElement.textContent = message;
    errorElement.classList.remove('hidden');
}

// Helper function to clear all errors
function clearErrors() {
    const errorElements = document.querySelectorAll('[id$="Error"]');
    errorElements.forEach(element => {
        element.classList.add('hidden');
        element.textContent = '';
    });
}

// Helper function to show status messages
function showStatus(message, isSuccess = false) {
    statusMessage.innerHTML = `
        <div class="p-3 rounded-md ${isSuccess ? 'bg-green-50 border border-green-200 text-green-700' : 'bg-red-50 border border-red-200 text-red-700'}">
            ${message}
        </div>
    `;
    statusMessage.classList.remove('hidden');
}

// Helper function to hide status messages
function hideStatus() {
    statusMessage.classList.add('hidden');
    statusMessage.innerHTML = '';
}

// Submit invite function
async function submitInvite(event) {
    event.preventDefault();

    // Clear previous status
    hideStatus();

    // Validate form
    if (!validate()) {
        return;
    }

    // Disable button and show loading state
    submitBtn.disabled = true;
    submitBtn.textContent = 'Sending...';

    try {
        // Prepare form data
        const formData = new FormData(form);
        const data = {
            sender_name: formData.get('sender_name') || '',
            sender_email: formData.get('sender_email'),
            recipient_email: formData.get('recipient_email'),
            title: formData.get('title') || '',
            datetime: formData.get('datetime'),
            meeting_link: formData.get('meeting_link') || '',
            message: formData.get('message') || ''
        };

        // Send POST request
        const response = await fetch('http://localhost:8080/api/invite', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
            },
            body: JSON.stringify(data)
        });

        if (response.ok) {
            showStatus('Meeting invite sent successfully! 🎉', true);
            form.reset();
        } else {
            const errorText = await response.text();
            showStatus(`Failed to send invite: ${errorText || 'Server error'}`);
        }
    } catch (error) {
        showStatus(`Network error: ${error.message}`);
    } finally {
        // Re-enable button
        submitBtn.disabled = false;
        submitBtn.textContent = 'Send Invite';
    }
}

// Add event listener to form
form.addEventListener('submit', submitInvite);

// Set minimum datetime to now
const now = new Date();
const localISOTime = new Date(now.getTime() - now.getTimezoneOffset() * 60000).toISOString().slice(0, 16);
document.getElementById('datetime').min = localISOTime;
