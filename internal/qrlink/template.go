package qrlink

import "html/template"

var previewTmpl = template.Must(template.New("preview").Parse(previewHTMLTemplate))
var downloadTmpl = template.Must(template.New("download").Parse(downloadHTMLTemplate))

const previewHTMLTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Airpay Payment QR</title>
    <style>
        * { box-sizing: border-box; margin: 0; padding: 0; }
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
            background: linear-gradient(135deg, #eff6ff 0%, #dbeafe 100%);
            min-height: 100vh;
            display: flex;
            align-items: center;
            justify-content: center;
            padding: 20px;
        }
        .card {
            background: white;
            border-radius: 20px;
            max-width: 400px;
            width: 100%;
            box-shadow: 0 20px 60px rgba(37,99,235,0.12);
            text-align: center;
            overflow: hidden;
        }
        .qr-bg-card {
            position: relative;
            width: 100%;
            display: block;
            line-height: 0;
        }
        .bg-img {
            width: 100%;
            height: auto;
            display: block;
        }
        .qr-overlay {
            position: absolute;
            top: 20.5%;
            left: 21%;
            width: 57%;
            line-height: 0;
        }
        .qr-overlay svg {
            width: 100%;
            height: auto;
            display: block;
        }
        .card-body {
            padding: 20px 24px 24px;
        }
        .details {
            background: #f8fafc;
            border-radius: 12px;
            padding: 14px 16px;
            margin-bottom: 20px;
            text-align: left;
        }
        .detail-row {
            display: flex;
            justify-content: space-between;
            align-items: center;
            padding: 5px 0;
            border-bottom: 1px solid #e2e8f0;
            font-size: 13px;
        }
        .detail-row:last-child { border-bottom: none; }
        .lbl { color: #6b7280; flex-shrink: 0; }
        .val { font-weight: 600; color: #1e293b; text-align: right; margin-left: 8px; word-break: break-all; }
        .val-amount { font-size: 18px; color: #059669; font-weight: 700; }
        .btn-upi {
            display: block;
            background: linear-gradient(135deg, #00588e 0%, #004a75 100%);
            color: white;
            text-decoration: none;
            padding: 14px 20px;
            border-radius: 12px;
            font-size: 15px;
            font-weight: 600;
            margin-bottom: 10px;
        }
        .btn-download {
            display: block;
            width: 100%;
            background: #f1f5f9;
            color: #475569;
            text-decoration: none;
            padding: 11px 20px;
            border-radius: 12px;
            font-size: 13px;
            font-weight: 500;
            margin-bottom: 16px;
            border: none;
            cursor: pointer;
            font-family: inherit;
        }
        .btn-download:hover {
            background: #e2e8f0;
        }
        .btn-download:active {
            background: #cbd5e1;
        }
        .hide { display: none; }
        .capture-hide { display: none !important; }
        .expiry { font-size: 11px; color: #94a3b8; }
        .expiry b { color: #f59e0b; }
        .footer { margin-top: 16px; font-size: 11px; color: #cbd5e1; }
    </style>
</head>
<body>
<div class="card">
    <div class="qr-bg-card">
        <img class="bg-img" src="/assets/images/qr_background.png" alt="Airpay QR Payment">
        <div class="qr-overlay">{{.QRSVG}}</div>
    </div>
    <div class="card-body">
    <div class="details">
        <div class="detail-row">
            <span class="lbl">Amount</span>
            <span class="val val-amount">&#8377;{{.Amount}}</span>
        </div>
        <div class="detail-row">
            <span class="lbl">Order ID</span>
            <span class="val">{{.OrderID}}</span>
        </div>
        <div class="detail-row">
            <span class="lbl">Transaction ID</span>
            <span class="val">{{.APTransactionID}}</span>
        </div>
        <div class="detail-row">
            <span class="lbl">Merchant ID</span>
            <span class="val">{{.MerchantID}}</span>
        </div>
    </div>
    <a href="{{.UPIDeepLink}}" class="btn-upi">&#128073; Open in UPI App</a>
    <button id="downloadBtn" class="btn-download">&#128248; Download QR Page (PNG)</button>
    <div class="expiry">
        Expires: <b id="expiryDate"></b><br>
        <span id="countdown" class="hide"></span>
    </div>
    <div class="footer">Powered by Airpay &middot; Works with Google Pay, PhonePe, Paytm &amp; any UPI app</div>
    </div>
</div>
<script>
// Live countdown timer
(function() {
    const expiryTimestamp = {{.ExpiryTimestamp}} * 1000; // Convert to milliseconds
    const expiryDate = new Date(expiryTimestamp);

    // Display exact expiry date/time
    const options = {
        year: 'numeric', month: 'short', day: 'numeric',
        hour: '2-digit', minute: '2-digit', second: '2-digit',
        hour12: true
    };
    document.getElementById('expiryDate').textContent = expiryDate.toLocaleString('en-IN', options);

    // Update countdown every second
    function updateCountdown() {
        const now = Date.now();
        const remaining = expiryTimestamp - now;

        if (remaining <= 0) {
            document.getElementById('countdown').innerHTML = '<span style="color:#ef4444;">&#9200; Expired</span>';
            document.getElementById('countdown').classList.remove('hide');
            return;
        }

        const hours = Math.floor(remaining / (1000 * 60 * 60));
        const mins = Math.floor((remaining % (1000 * 60 * 60)) / (1000 * 60));
        const secs = Math.floor((remaining % (1000 * 60)) / 1000);

        document.getElementById('countdown').innerHTML =
            '<span style="color:#f59e0b;">&#9201; ' + hours + 'h ' + mins + 'm ' + secs + 's remaining</span>';
        document.getElementById('countdown').classList.remove('hide');
    }

    updateCountdown();
    setInterval(updateCountdown, 1000);
})();
</script>
<script src="/assets/js/html-to-image.min.js"></script>
<script>
document.getElementById('downloadBtn').addEventListener('click', function() {
    const btn = this;
    const originalText = btn.innerHTML;
    btn.disabled = true;
    btn.innerHTML = '&#9203; Generating...';

    const card = document.querySelector('.card');
    const captureHideEls = card.querySelectorAll('.btn-upi, #downloadBtn, .expiry, .footer');
    captureHideEls.forEach(el => el.classList.add('capture-hide'));

    htmlToImage.toPng(card, {
        backgroundColor: '#ffffff',
        pixelRatio: 3,
        skipFonts: false,
        cacheBust: true
    }).then(dataUrl => {
        captureHideEls.forEach(el => el.classList.remove('capture-hide'));
        const link = document.createElement('a');
        link.download = 'airpay-qr-{{.APTransactionID}}.png';
        link.href = dataUrl;
        link.click();

        btn.innerHTML = originalText;
        btn.disabled = false;
    }).catch(err => {
        captureHideEls.forEach(el => el.classList.remove('capture-hide'));
        console.error('PNG generation failed:', err);
        btn.innerHTML = '&#10060; Failed - Try again';
        setTimeout(() => {
            btn.innerHTML = originalText;
            btn.disabled = false;
        }, 2000);
    });
});
</script>
</body>
</html>`

const downloadHTMLTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Airpay Payment QR - {{.APTransactionID}}</title>
    <style>
        * { box-sizing: border-box; margin: 0; padding: 0; }
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
            background: white;
            min-height: 100vh;
            display: flex;
            align-items: center;
            justify-content: center;
            padding: 20px;
        }
        .card {
            background: white;
            border: 2px solid #e5e7eb;
            border-radius: 16px;
            max-width: 400px;
            width: 100%;
            text-align: center;
            overflow: hidden;
        }
        .qr-bg-card {
            position: relative;
            width: 100%;
            display: block;
            line-height: 0;
        }
        .bg-img {
            width: 100%;
            height: auto;
            display: block;
        }
        .qr-overlay {
            position: absolute;
            top: 20.5%;
            left: 21%;
            width: 57%;
            line-height: 0;
        }
        .qr-overlay img {
            width: 100%;
            height: auto;
            display: block;
        }
        .card-body {
            padding: 20px 24px 24px;
        }
        .details {
            background: #f8fafc;
            border-radius: 12px;
            padding: 14px 16px;
            margin-bottom: 16px;
            text-align: left;
        }
        .detail-row {
            display: flex;
            justify-content: space-between;
            align-items: center;
            padding: 5px 0;
            border-bottom: 1px solid #e2e8f0;
            font-size: 13px;
        }
        .detail-row:last-child { border-bottom: none; }
        .lbl { color: #6b7280; flex-shrink: 0; }
        .val { font-weight: 600; color: #1e293b; text-align: right; margin-left: 8px; word-break: break-all; }
        .val-amount { font-size: 18px; color: #059669; font-weight: 700; }
        .footer { margin-top: 16px; font-size: 11px; color: #94a3b8; }
        @media print {
            body { background: white; }
            .card { border: none; box-shadow: none; }
        }
    </style>
</head>
<body>
<div class="card">
    <div class="qr-bg-card">
        <img class="bg-img" src="/assets/images/qr_background.png" alt="Airpay QR Payment">
        <div class="qr-overlay"><img id="qrImage" alt="UPI QR Code"></div>
    </div>
    <div class="card-body">
    <div class="details">
        <div class="detail-row">
            <span class="lbl">Amount</span>
            <span class="val val-amount">&#8377;{{.Amount}}</span>
        </div>
        <div class="detail-row">
            <span class="lbl">Order ID</span>
            <span class="val">{{.OrderID}}</span>
        </div>
        <div class="detail-row">
            <span class="lbl">Transaction ID</span>
            <span class="val">{{.APTransactionID}}</span>
        </div>
        <div class="detail-row">
            <span class="lbl">Merchant ID</span>
            <span class="val">{{.MerchantID}}</span>
        </div>
    </div>
    <div class="footer">Powered by Airpay &middot; Works with Google Pay, PhonePe, Paytm &amp; any UPI app</div>
    </div>
</div>
<script>
// Set QR image source on document load
(function() {
    const QRImageBase64 = '{{.QRImageBase64}}';
    document.getElementById('qrImage').src = QRImageBase64;
})();
</script>
</body>
</html>`
