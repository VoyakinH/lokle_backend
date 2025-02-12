package mailer

import (
	"fmt"

	"github.com/VoyakinH/lokle_backend/config"
	"github.com/sirupsen/logrus"
	"gopkg.in/gomail.v2"
)

func SendVerifiedEmail(to_email string, first_name string, second_name string, token string) error {
	msg := gomail.NewMessage()
	msg.SetHeader("From", config.Mailer.Email)
	msg.SetHeader("To", to_email)
	msg.SetHeader("Subject", "Подтвержение почты Столичный-КИТ")
	msg.SetBody("text/html", fmt.Sprintf("Приветствуем, %s %s! <br/> Для подтверждения электронной почты, пройдите, пожалуйста, по ссылке: <br/>  https://kit.lokle.ru/login?verification_email_token=%s <br/> Если Вы получили это письмо по ошибке, просто игнорируйте его. <br/> Ссылка активна в течение 7 дней.", first_name, second_name, token))

	n := gomail.NewDialer("mail.s-kit.moscow", 25, config.Mailer.Email, config.Mailer.Password)

	if err := n.DialAndSend(msg); err != nil {
		logrus.Errorf("Mailer.SendVerefiedEmail: failed send email via kit smpt server with err: %s", err)
		msg.SetHeader("From", config.Mailer.AdditionalEmail)
		n = gomail.NewDialer("smtp.mail.ru", 465, config.Mailer.AdditionalEmail, config.Mailer.AdditionalPassword)
		if err := n.DialAndSend(msg); err != nil {
			return err
		}
	}
	return nil
}

func SendCompleteChildRegistrationEmail(to_email string, first_name string, second_name string, password string) error {
	msg := gomail.NewMessage()
	msg.SetHeader("From", config.Mailer.Email)
	msg.SetHeader("To", to_email)
	msg.SetHeader("Subject", "Регистрация Столичный-КИТ")
	msg.SetBody("text/html", fmt.Sprintf("Приветствуем, %s %s! <br/> Ваши данные для входа: <br/>  Логин: %s <br/> Пароль: %s <br/> Если Вы получили это письмо по ошибке, просто игнорируйте его. <br/>", first_name, second_name, to_email, password))

	n := gomail.NewDialer("mail.s-kit.moscow", 25, config.Mailer.Email, config.Mailer.Password)

	if err := n.DialAndSend(msg); err != nil {
		logrus.Errorf("Mailer.SendVerefiedEmail: failed send email via kit smpt server with err: %s", err)
		msg.SetHeader("From", config.Mailer.AdditionalEmail)
		n = gomail.NewDialer("smtp.mail.ru", 465, config.Mailer.AdditionalEmail, config.Mailer.AdditionalPassword)
		if err := n.DialAndSend(msg); err != nil {
			return err
		}
	}
	return nil
}
