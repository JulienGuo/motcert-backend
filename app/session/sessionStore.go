package session

import (
	"time"
)

type SessionStore struct {
	sid          string                      //session id唯一标示
	timeAccessed time.Time                   //最后访问时间
	value        map[interface{}]interface{} //session里面存储的值
}

func (st *SessionStore) Set(key, value interface{}) error {
	st.value[key] = value
	return pder.SessionUpdate(st.sid)
}

func (st *SessionStore) Get(key interface{}) interface{} {
	err := pder.SessionUpdate(st.sid)
	if err == nil {
		if v, ok := st.value[key]; ok {
			return v
		} else {
			return nil
		}
	} else {
		return err
	}
}

func (st *SessionStore) Delete(key interface{}) error {
	delete(st.value, key)
	return pder.SessionUpdate(st.sid)
}

func (st *SessionStore) SessionID() string {
	return st.sid
}
