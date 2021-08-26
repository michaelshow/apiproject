package controllers

import "github.com/astaxie/beego"

// AccountController 用户登录与注册
type AccountController struct {
	beego.Controller
}

func (a *AccountController) Login() {
	userName := a.Ctx.Input.Param(":userName")
	if userName != "" {
		v := map[string]string{"name": userName, "age": "18"}
		a.Data["json"] = v
	}
	v := map[string]string{"name": userName, "age": "19"}
	a.Data["json"] = v
	a.ServeJSON()
}
