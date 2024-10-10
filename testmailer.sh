#!/bin/bash

# set the sender and recipient email addresses
sender="support@ammer.io"
recipient="sidohin.felix@gmail.com"

# set the subject and body of the email
subject="Test Email with Attachment"
body="This is a test email with attachment."

# set the attachment file name and path
filename="test.pdf"
filepath="/Users/felixsidokhine/Desktop/FORM_902.9E.pdf"

# create a boundary string
boundary=$(uuidgen)

# build the email message body
{
  echo "From: ${sender}"
  echo "To: ${recipient}"
  echo "Subject: ${subject}"
  echo "MIME-Version: 1.0"
  echo "Content-Type: multipart/mixed; boundary=${boundary}"
  echo ""
  echo "--${boundary}"
  echo "Content-Type: text/plain; charset=UTF-8"
  echo "Content-Disposition: inline"
  echo ""
  echo "${body}"
  echo ""
  echo "--${boundary}"
  echo "Content-Type: application/octet-stream"
  echo "Content-Disposition: attachment; filename=${filename}"
  echo ""
  cat "${filepath}"
  echo ""
  echo "--${boundary}--"
} | /usr/sbin/sendmail -t

