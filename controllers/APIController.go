package controllers

import (
	"alertCenter/core"
	"alertCenter/core/db"
	"alertCenter/core/service"
	"alertCenter/core/user"
	"alertCenter/models"
	"alertCenter/util"
	"encoding/json"
	"strconv"
	"time"

	"github.com/astaxie/beego"
	"gopkg.in/mgo.v2/bson"
)

type APIController struct {
	//beego.Controller
	APIBaseController
	session      *db.MongoSession
	alertService *service.AlertService
	teamServcie  *service.TeamService
}

func (e *APIController) Receive() {
	data := e.Ctx.Input.RequestBody
	if data != nil && len(data) > 0 {
		var Alerts []*models.Alert = make([]*models.Alert, 0)
		err := json.Unmarshal(data, &Alerts)
		if err == nil {
			core.HandleAlerts(Alerts)
			e.Data["json"] = util.GetSuccessJson("receive alert success")
		} else {
			e.Data["json"] = util.GetErrorJson("receive a unknow data")
		}
	} else {
		e.Data["json"] = util.GetErrorJson("receive a unknow data")
	}

	e.ServeJSON()
}

// func (e *APIController) AddTag() {
// 	we := &core.notice.WeAlertSend{}
// 	if ok := we.GetAllTags(); ok {
// 		e.Data["json"] = util.GetSuccessJson("get weiTag success")
// 	} else {
// 		e.Data["json"] = util.GetFailJson("get weiTag faild")
// 	}
// 	e.ServeJSON()
// }

func (e *APIController) HandleAlert() {
	ID := e.GetString(":ID")
	Type := e.GetString(":type")
	message := e.GetString("message")
	if len(ID) == 0 || len(Type) == 0 {
		e.Data["json"] = util.GetErrorJson("参数格式错误")
	} else {
		session := db.GetMongoSession()
		if session != nil {
			defer session.Close()
		}
		alertService := service.GetAlertService(session)
		alert := alertService.FindByID(ID)
		if alert == nil {
			e.Data["json"] = util.GetFailJson("报警信息不存在，id信息错误")
		} else {
			if Type == "handle" {
				alert.IsHandle = 1
			} else if Type == "miss" {
				alert.IsHandle = -1
			}
			alert.HandleDate = time.Now()
			alert.HandleMessage = message
			if ok := alertService.Update(alert); ok {
				e.Data["json"] = util.GetSuccessJson("登记成功")
			} else {
				e.Data["json"] = util.GetFailJson("登记失败")
			}
		}
	}
	e.ServeJSON()
}

func (e *APIController) GetAlerts() {
	pageSizeStr := e.Ctx.Request.FormValue("pageSize")
	pageSize, err := strconv.Atoi(pageSizeStr)
	if err != nil {
		pageSize = 20
	}

	pageStr := e.Ctx.Request.FormValue("page")
	page, err := strconv.Atoi(pageStr)
	if err != nil {
		page = 1
	}

	receiver := e.APIBaseController.Username

	// if admin. Should show all the alerts.
	//user, err := gitlab.GetUserByUsername(receiver)
	relation := user.Relation{}
	user := relation.GetUserByName(receiver)
	if err != nil {
		beego.Error(err)
	} else if user.IsAdmin {
		receiver = "all"
	}

	e.session = db.GetMongoSession()
	if e.session != nil {
		defer e.session.Close()
	}

	if e.session == nil {
		e.Data["json"] = util.GetFailJson("get database session faild.")
		goto over
	} else {
		e.alertService = service.GetAlertService(e.session)
		if len(receiver) != 0 && receiver != "all" {
			alerts := e.alertService.FindByUser(receiver, pageSize, page)
			beego.Info("Get", len(alerts), " alerts")
			if alerts == nil {
				e.Data["json"] = util.GetFailJson("get database collection faild or receiver is error ")
				goto over
			} else {
				e.Data["json"] = util.GetSuccessReJson(alerts)
				goto over
			}
		} else if receiver == "all" {
			alerts := e.alertService.FindAll(pageSize, page)
			if alerts == nil {
				e.Data["json"] = util.GetFailJson("get database collection faild")
				goto over
			} else {
				e.Data["json"] = util.GetSuccessReJson(alerts)
				goto over
			}
		} else {
			e.Data["json"] = util.GetErrorJson("api use error,please provide receiver")
			goto over
		}
	}
over:
	e.ServeJSON()
}

//SetNoticeMode 控制是否发送邮件
func (e *APIController) SetNoticeMode() {
	userName := e.Ctx.Input.Header("user")
	status, err := e.GetBool(":status")
	if err != nil {
		e.Data["json"] = util.GetFailJson("api error,status not provide")
	} else {
		relation := user.Relation{}
		user := relation.GetUserByName(userName)
		if user == nil || !user.IsAdmin {
			e.Data["json"] = util.GetFailJson("Do not allow the operation")
		} else {
			session := db.GetMongoSession()
			if session != nil {
				defer session.Close()
			}
			if session == nil {
				e.Data["json"] = util.GetErrorJson("get mongo session error when init NoticeOn ")
			} else {
				service := &service.GlobalConfigService{
					Session: session,
				}
				config := service.GetConfig("noticeOn")
				if config == nil {
					config = &models.GlobalConfig{}
					config.Name = "noticeOn"
					config.Value = status
					config.AddTime = time.Now()
					session.Insert("GlobalConfig", config)
				} else {
					config.Value = status
					service.Update(config)
				}
				e.Data["json"] = util.GetSuccessJson("noticeon update success")
			}
		}
	}
	e.ServeJSON()
}

func (e *APIController) GetNoticeMode() {
	userName := e.Ctx.Input.Header("user")
	relation := user.Relation{}
	user := relation.GetUserByName(userName)
	if user == nil || !user.IsAdmin {
		e.Data["json"] = util.GetFailJson("Do not allow the operation")
	} else {
		session := db.GetMongoSession()
		if session != nil {
			defer session.Close()
		}
		if session == nil {
			e.Data["json"] = util.GetErrorJson("get mongo session error when init NoticeOn ")
		} else {
			service := &service.GlobalConfigService{
				Session: session,
			}
			config := service.GetConfig("noticeOn")
			if config == nil {
				config = &models.GlobalConfig{}
				config.Name = "noticeOn"
				config.Value = true
				config.AddTime = time.Now()
				session.Insert("GlobalConfig", config)

			}
			e.Data["json"] = util.GetSuccessReJson(config)
		}
	}
	e.ServeJSON()
}

func (e *APIController) AddTrustIP() {

	userName := e.Ctx.Input.Header("user")
	relation := user.Relation{}
	user := relation.GetUserByName(userName)
	if user == nil || !user.IsAdmin {
		e.Data["json"] = util.GetFailJson("Do not allow the operation")
	} else {
		data := e.Ctx.Input.RequestBody
		var config = models.GlobalConfig{}
		err := json.Unmarshal(data, &config)
		if err != nil {
			e.Data["json"] = util.GetErrorJson("data parse error")
		} else {
			session := db.GetMongoSession()
			if session != nil {
				defer session.Close()
			}
			if session == nil {
				e.Data["json"] = util.GetErrorJson("get mongo session error when add trust ip ")
			} else {
				service := &service.GlobalConfigService{
					Session: session,
				}
				if ok := service.CheckExist("TrustIP", config.Value); !ok {
					service.Session.Insert("GlobalConfig", &models.GlobalConfig{
						Name:    "TrustIP",
						Value:   config.Value,
						AddTime: time.Now(),
						ID:      bson.NewObjectId(),
					})
				}
				re := service.GetConfigA("TrustIP", config.Value)
				if re != nil {
					e.Data["json"] = util.GetSuccessReJson(re)
				} else {
					e.Data["json"] = util.GetFailJson("insert trust ip faild")
				}

			}
		}
	}
	e.ServeJSON()
}

func (e *APIController) GetTrustIP() {
	userName := e.Ctx.Input.Header("user")
	relation := user.Relation{}
	user := relation.GetUserByName(userName)
	if user == nil || !user.IsAdmin {
		e.Data["json"] = util.GetFailJson("Do not allow the operation")
	} else {
		session := db.GetMongoSession()
		if session != nil {
			defer session.Close()
		}
		if session == nil {
			e.Data["json"] = util.GetErrorJson("get mongo session error when add trust ip ")
		} else {
			service := &service.GlobalConfigService{
				Session: session,
			}

			re := service.GetAllConfig("TrustIP")
			if re != nil {
				e.Data["json"] = util.GetSuccessReJson(re)
			} else {
				e.Data["json"] = util.GetFailJson("there is not trust ip.")
			}

		}
	}
	e.ServeJSON()
}
func (e *APIController) DeleteTrustIP() {
	userName := e.Ctx.Input.Header("user")
	relation := user.Relation{}
	user := relation.GetUserByName(userName)
	if user == nil || !user.IsAdmin {
		e.Data["json"] = util.GetFailJson("Do not allow the operation")
	} else {
		ID := e.GetString(":ID")
		if ID == "" {
			e.Data["json"] = util.GetErrorJson("Trust ip id is not provide")
		} else {
			session := db.GetMongoSession()
			if session != nil {
				defer session.Close()
			}
			if session == nil {
				e.Data["json"] = util.GetErrorJson("get mongo session error when add trust ip ")
			} else {
				service := &service.GlobalConfigService{
					Session: session,
				}
				re := service.DeleteByID(ID)
				if re {
					e.Data["json"] = util.GetSuccessJson("remove trust ip success")
				} else {
					e.Data["json"] = util.GetFailJson("get trust ip faild")
				}

			}
		}
	}
	e.ServeJSON()
}

//RefreshCache 更新缓存的用户和组信息app信息
func (e *APIController) RefreshCache() {
	userName := e.Ctx.Input.Header("user")
	relation := user.Relation{}
	user := relation.GetUserByName(userName)
	if user != nil && user.IsAdmin {
		relation.RefreshCache()
		e.Data["json"] = util.GetSuccessJson("fresh cache success")
	} else {
		e.Data["json"] = util.GetFailJson("Do not allow the operation")
	}
	e.ServeJSON()
}
