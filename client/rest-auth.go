package client

/*
// The jwt middleware.
var authMiddleware = GinJWTMiddleware{
	Realm:         "test zone",
	Key:           []byte("secret key"), // config
	Timeout:       time.Hour,
	MaxRefresh:    time.Hour,
	Authenticator: authenticator,
	// Authorizator:  authorizator,
	Unauthorized: unauthorized,

	TokenLookup: "header:Authorization",

	TokenHeadName: "Bearer",

	TimeFunc: time.Now,
}

var authenticator = func(userId string, deviceId string, password string, c *gin.Context) (appuser.User, bool) {

	query := bson.M{"userID": userId}

	user := appuser.User{}

	err := usersData.Find(query).One(&user)

	if err != nil || len(user.UserID) == 0 {
		return appuser.User{}, false
	}

	return user, true

}

// var authorizator = func(userId string, c *gin.Context) bool {
// 	if userId == "admin" {
// 		return true
// 	}
//
// 	return false
// }

var unauthorized = func(c *gin.Context, code int, message string) {
	c.JSON(code, gin.H{
		"code":    code,
		"message": message,
	})
}

/*
// LoginHandler can be used by clients to get a jwt token.
// Payload needs to be json in the form of {"username": "USERNAME", "password": "PASSWORD", "deviceid": "DEVICEID"}.
// Reply will be of the form {"token": "TOKEN"}.
func (mw *GinJWTMiddleware) LoginHandler(c *gin.Context) {

	// Initial middleware default setting.
	mw.MiddlewareInit()

	var loginVals Login

	if c.ShouldBindWith(&loginVals, binding.JSON) != nil {
		mw.unauthorized(c, http.StatusBadRequest, "Missing Username, Password or DeviceID")
		return
	}

	if mw.Authenticator == nil {
		mw.unauthorized(c, http.StatusInternalServerError, "Missing define authenticator func")
		return
	}

	user, ok := mw.Authenticator(loginVals.Username, loginVals.DeviceID, loginVals.Password, c) // user can be empty
	userID := user.UserID

	if len(userID) == 0 {
		ok = false
	}

	// Create the token
	token := jwt.New(jwt.GetSigningMethod(mw.SigningAlgorithm))
	claims := token.Claims.(jwt.MapClaims)
	if mw.PayloadFunc != nil {
		for key, value := range mw.PayloadFunc(loginVals.Username) {
			claims[key] = value
		}
	}

	if userID == "" {
		userID = loginVals.Username
	}

	expire := mw.TimeFunc().Add(mw.Timeout)
	claims["id"] = userID
	claims["exp"] = expire.Unix()
	claims["orig_iat"] = mw.TimeFunc().Unix()

	tokenString, err := token.SignedString(mw.Key)
	if err != nil {
		mw.unauthorized(c, http.StatusUnauthorized, "Create JWT Token faild")
		return
	}

	// If user auths with DeviceID and UserID that
	// already exists in DB we refresh JWT token.
	for _, concreteDevice := range user.Devices {

		// case of expired token on device or relogin from same device with
		// userID and deviceID existed in DB
		if concreteDevice.DeviceID == loginVals.DeviceID {
			sel := bson.M{"userID": user.UserID, "devices.JWT": concreteDevice.JWT}
			update := bson.M{"$set": bson.M{"devices.$.JWT": tokenString}}
			err = usersData.Update(sel, update)
			responseErr(c, err, http.StatusInternalServerError) // 500
		} else {
			// case of adding new device to user account
			// e.g. user want to use app on another device

			// BUG: cant crate new device -- fixed
			device := createDevice(loginVals.DeviceID, c.ClientIP(), tokenString)
			user.Devices = append(user.Devices, device)

			// IDEA: speed improvmet: refresh several fields but not full object.
			sel := bson.M{"userID": userID}
			err = usersData.Update(sel, user)

			responseErr(c, err, http.StatusInternalServerError) // 500

		}
	}

	if !ok {
		device := createDevice(loginVals.DeviceID, c.ClientIP(), tokenString)

		var wallet []appuser.Wallet
		var devices []appuser.Device
		devices = append(devices, device)

		newUser := createUser(loginVals.Username, devices, wallet)

		user = newUser

		err = usersData.Insert(user)
		responseErr(c, err, http.StatusInternalServerError) // 500

	}

	c.JSON(http.StatusOK, gin.H{
		"token":  tokenString,
		"expire": expire.Format(time.RFC3339),
	})
}

func createUser(userid string, device []appuser.Device, wallets []appuser.Wallet) appuser.User {
	return appuser.User{
		UserID:  userid,
		Devices: device,
		Wallets: wallets,
	}
}

func createDevice(deviceid, ip, jwt string) appuser.Device {
	return appuser.Device{
		DeviceID:       deviceid,
		JWT:            jwt,
		LastActionIP:   ip,
		LastActionTime: time.Now(),
	}
}
*/
func createWallet(chain, address, addressID string) Wallet {
	return Wallet{
		Chain: chain,
		Adresses: []Address{
			Address{
				Address:   address,
				AddressID: addressID,
			},
		},
	}
}
