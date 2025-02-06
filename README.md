# sendmail

A simple bulk mail sender with random timeouts to prevent mailservers from flagging your account of spam usage.

## How to use

Before running configure your environment. Copy the ```.env.example``` file to ```.env``` and add your email configuration.

```bash
make && nohup ./sendmail --csv receipiants.csv --html email.html > nohup.log &
```

## Disclaimer

This script is provided for ethical and responsible use only. It is intended for legitimate purposes such as sending bulk emails to subscribers, customers, or contacts who have given their consent.

By using this script, you agree to comply with all applicable laws, regulations, and best practices regarding email communication, including but not limited to anti-spam laws (such as GDPR, CAN-SPAM, and other relevant regulations).

I am not liable for any misuse, abuse, or consequences resulting from the use of this script. It is your responsibility to ensure that all emails sent using this tool adhere to ethical and legal standards.

Use it wisely and with good intentions.
