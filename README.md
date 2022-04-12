# Gobrut

Gobrut is a tool to perform brute force web form attacks. It currently supports json and form bodies.

## Usage

    go run main.go -t <url> -u <usernames.txt> -p <passwords.txt> -m <json|form> -n <usernameField> -s <passwordField> -f <failText> -b <extraBody>

 - Usernames and passwords must be plain lists
 - The usernameField and passwordField refers to the field names in the request body
 - Extra body is in the form of key=value&key2=value2 (it will be added in the proper format, be it form or json)

## Example

Jenkins:

    gobrut -t http://127.0.0.1:8080/j_spring_security_check -u default-users.txt -p .default.txt -m form -n j_username -s j_password -b from=&Submit=Sign+in -f "Invalid username or password"

    Successfully logged in:  root password

## License

MIT