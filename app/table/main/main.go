package main

import (
	"fmt"
	"github.com/coschain/contentos-go/app/table"
	"github.com/coschain/contentos-go/db/storage"
	"github.com/coschain/contentos-go/prototype"
	"log"
)

func main() {
	/*
	  ------------------------------------------------
	  type,pName,mKey,unique,sort,reverseSort
	  **the csv file Field  Explanation
	  type: the data type of property in table
	  pName: the name of property in table
	  mKey: whether the property is a primary key
	  unique: whether the property is support unique query (0:not support 1:support)
	  sort: whether the property is support sort by order when query (0:not support 1:support)
	  reverseSort:whether the property is support sort by reverse order when query (0:not support 1:support)
	  ------------------------------------------------
	*/

	//db, _ := storage.NewLevelDatabase("/Users/yykingking/abc123.db")
	db, err := storage.NewDatabase("./demos/pbTool.db")
	if err != nil {
		return
	}
	db.Start(nil)
	//defer db.Close()

	//1.create the table wrap
	//we can use the type  which is contained in another created pb struct,
	// such as "prototype.account_name" in AccountName 、prototype.time_point_sec
	//MakeXXX func can create a pb struct
	mKey := prototype.NewAccountName("myName")
	wrap := table.NewSoDemoWrap(db, mKey)
	if wrap == nil {
		//crreate fail , the db already contain table with current mainKey
		log.Println("crreate fail , the db already contain table with current mainKey")
		return
	}
	if wrap.CheckExist() {
		wrap.RemoveDemo()
	}
	//2.save table data to db
	err = wrap.Create(func(tInfo *table.SoDemo) {
		tInfo.Owner = mKey
		tInfo.Title = "hello"
		tInfo.Content = "test the pb tool"
		tInfo.Idx = 1001
		tInfo.LikeCount = 100
		tInfo.Taglist = []string{"#NBA"}
		tInfo.ReplayCount = 100
		tInfo.PostTime = creTimeSecondPoint(20120401)
	})
	if err != nil {
		fmt.Printf("create new table of Demo fail,the error is %s \n",err)
		return
	}

	err = wrap.Md(func(tInfo *table.SoDemo) {
		tInfo.Idx = 1100
		tInfo.LikeCount = 10
		tInfo.Owner = prototype.NewAccountName("test")
		tInfo.Title = "test md title"
		tInfo.PostTime = nil
	})

	if err != nil {
		fmt.Printf("Test: md idx fail, the error is %v \n", err)
	}
    likeCount := wrap.GetLikeCount()
    fmt.Printf("the modified liekcount is %v \n", likeCount)

    fmt.Printf("the modified title is %v \n", wrap.GetTitle())

	key1 := prototype.NewAccountName("myName1")
	wrap1 := table.NewSoDemoWrap(db, key1)
	if wrap1 == nil {
		//crreate fail , the db already contain table with current mainKey
		log.Println("create fail , the db already contain table with current mainKey myName1")
		return
	}
	if wrap1.CheckExist() {
		fmt.Println("remove the wrap")
		res := wrap1.RemoveDemo()
		fmt.Printf("remove result is %v \n", res)
	}
	err = wrap1.Create(func(tInfo *table.SoDemo) {
		tInfo.Owner = key1
		tInfo.Title = "hello1"
		tInfo.Content = "wrap1"
		tInfo.Idx = 1002
		tInfo.LikeCount = 200
		tInfo.Taglist = []string{"#Car"}
		tInfo.ReplayCount = 150
		tInfo.PostTime = creTimeSecondPoint(20120403)
	})
	if err != nil {
		fmt.Printf("create new table of Demo fail,the error is %s \n",err)
	}
	con := wrap1.GetContent()
	fmt.Printf("the content of new wrap is %s \n",con)
	idx1 := wrap1.GetIdx()
	fmt.Printf("the idx of new wrap is %d \n", idx1)
	fmt.Printf("the likeCount of new wrap is %d \n", wrap1.GetLikeCount())
	/*
	   --------------------------
	   Get Property（the GetXXX function  return the property value）
	   --------------------------*/

	//get title
	t := wrap.GetTitle()
	if t != "" {
		fmt.Printf("the title is %s \n", t)
	} else {
		fmt.Printf("get title fail")
	}

	//get content
	c := wrap.GetContent()
	if c != "" {
		fmt.Printf("the content is %s \n", c)
	} else {
		fmt.Println("get content fail")
	}
	tMdErr := wrap.Md(func(tInfo *table.SoDemo) {
		tInfo.Content = "hello world"
	})
	if tMdErr != nil {
		fmt.Println("modify content fail")
	}
   fmt.Printf("The modified content is %v \n", wrap.GetContent())

	/*
	  --------------------------
	   Modify property value (******can't modify the mainkey)
	  --------------------------*/
	//modify content
	cMdRes := wrap.Md(func(tInfo *table.SoDemo) {
		tInfo.Content = "test md the content"
	})
	if cMdRes != nil {
		fmt.Println("modify content fail")
	} else {
		fmt.Printf("The modified content is %v \n", wrap.GetContent())
	}

	tMdRes := wrap.Md(func(tInfo *table.SoDemo) {
		tInfo.Taglist = []string{"#Football"}
	})
	if tMdRes != nil {
		fmt.Println("modify taglist fail")
	} else {
		tag := wrap.GetTaglist()
		fmt.Printf("the modified taglist is %v \n", tag)
	}

	/*--------------------------
	  Sort Query List
	 --------------------------*/
	//1.create the sort wrap for property which is support sorting (E.g postTime)
	tSortWrap := table.SDemoPostTimeWrap{}
	tSortWrap.Dba = db
	//2.start query data of range(sort by order)
	//start = nil  end = nil (query the db from start to end)
	//start = nil (query from start the db)
	//end = nil (query to the end of db)
	//maxCount represent the maximum amount of data you want to get，if the maxCount is greater than or equal to
	//the total count of data in result,traverse all data;otherwise traverse part of the data
	//if query by order the start value can't greater than end value
	err = tSortWrap.ForEachByOrder(creTimeSecondPoint(20120401),
		creTimeSecondPoint(20120415),nil ,nil,func(mVal *prototype.AccountName, sVal *prototype.TimePointSec,
			idx uint32) bool {
			//we can get the main key and sub key from the callBack
			if mKey == nil {
				fmt.Println("get main key fail")
			} else {
				fmt.Printf("the main key is %s in range \n", mKey.Value)
			}

			if sVal == nil {
				fmt.Println("get postTime fail")
			} else {
				fmt.Printf("the postTime is %d \n", sVal.UtcSeconds)
			}
			//if return true,continue iterating until the end iteration;otherwise stop iteration immediately
			if mKey.Value == "myName" {
				return false
			}
			return true
		})
	if err != nil {
		fmt.Printf("QueryList by order fail,the error is %s \n",err)
	}
	
	//query by reverse order
	//start = nil  end = nil (query the db from start to end)
	//start = nil (query from start the db)
	//end = nil (query to the end of db)
	//if query by reverse order the start value can't less than end value
	err = tSortWrap.ForEachByRevOrder(creTimeSecondPoint(20120415),
		creTimeSecondPoint(20120401),nil,nil, func(mVal *prototype.AccountName, sVal *prototype.TimePointSec,
			idx uint32) bool {
			if mVal == nil {
				fmt.Println("query by reverse order get main key fail")
			} else {
				fmt.Printf("the main key is %s in reverse order  \n", mVal.Value)
			}
			if sVal == nil {
				fmt.Println("query by reverse order get postTime fail")
			} else {
				fmt.Printf("the postTime is %d in reverse order \n", sVal.UtcSeconds)
			}
			if idx < 200 {
				return true
			}
			return false
		})
	if err != nil {
		fmt.Printf("Query data in reverse order fail,the error is %s \n",err)
	}

	//query without start
	err = tSortWrap.ForEachByOrder(nil, creTimeSecondPoint(20120422),nil,nil,
		func(mVal *prototype.AccountName, sVal *prototype.TimePointSec, idx uint32) bool {
			if mVal == nil {
				fmt.Println("get main key fail in range when query without start 1111")
			} else {
				fmt.Printf("the main key is %s in range when query without start  \n", mVal.Value)
			}
			if idx < 100 {
				return true
			}
			return false
	})
	if err != nil {
		fmt.Printf("Query list without start fail, the error is %s  \n",err)
	}

	//query without end
	err = tSortWrap.ForEachByOrder(creTimeSecondPoint(20120000), nil,nil,nil,
		func(mVal *prototype.AccountName, sVal *prototype.TimePointSec, idx uint32) bool  {
			if mVal == nil {
				fmt.Println("get main key fail in range when query without end")
			} else {
				fmt.Printf("the main key is %s in range when query without end \n", mVal.Value)
			}
			return true
		})
	if err != nil {
		fmt.Printf("Query list without end fail, the error is %s  \n",err)
	}

	//query without start and end
	err = tSortWrap.ForEachByOrder(nil, nil,nil,nil,
		func(mVal *prototype.AccountName, sVal *prototype.TimePointSec, idx uint32) bool {
			if mVal == nil {
				fmt.Println("get main key fail in range when query without start and end")
			} else {
				fmt.Printf("the main key is %s when query without start and end  \n", mVal.Value)
			}
			return true
	})
	if err != nil {
		fmt.Printf("Query list without start and end fail, the error is %s  \n",err)
	}


	//query without start and end by reverse order
	err = tSortWrap.ForEachByRevOrder(nil, nil,nil,nil,
		func(mVal *prototype.AccountName, sVal *prototype.TimePointSec, idx uint32) bool {
			if mVal == nil {
				fmt.Println("get main key fail in range when query without start and end by reverse sort ")
			} else {
				fmt.Printf("the main key is %s in range when query without start and end by reverse sort \n",
					mVal.Value)
			}
			if idx < 100 {
				return true
			}
			return false
	})
	if err != nil {
		fmt.Printf("Query list in reverse order without start and end fail, the error is %s  \n",err)
	}

	/*
	 --------------------------
	  unique Query List (only support query the property which is flag unique)
	 --------------------------*/
	//1.create the uni wrap of property which is need unique query
	var idx int64 = 1100
	//create the UniXXXWrap
	uniWrap := table.UniDemoIdxWrap{}
	//set the dataBase to UniXXXWrap
	uniWrap.Dba = db
	//2.use UniQueryXX func to query data meanWhile return the table wrap
	dWrap := uniWrap.UniQueryIdx(&idx)
	if dWrap == nil {
		fmt.Printf("uni query fail \n")
	} else {
		title := dWrap.GetTitle()
		fmt.Printf("the title of index %v is %s \n", idx,title)
	}

	//unique query mainkey(E.g query owner)
	mUniWrap := table.UniDemoOwnerWrap{}
	mUniWrap.Dba = db
	str := "myName"
	res := mUniWrap.UniQueryOwner(prototype.NewAccountName(str))
	if res != nil {
		fmt.Printf("owner is %s,the idx is %d \n",str,res.GetIdx())
	}

	/*
	   remove tabale data from db
	*/
	//judge the table of current mainKey if is exist
	isExsit := wrap.CheckExist()
	if isExsit {
		res := wrap.RemoveDemo()
		if !res {
			fmt.Println("remove the table data fail")
		}
	}
	isExsit = wrap1.CheckExist()
	if isExsit {
		res := wrap1.RemoveDemo()
		if !res {
			fmt.Println("remove the table1 data fail")
		}
	}
	db.Close()
}

func creTimeSecondPoint(t uint32) *prototype.TimePointSec {
	val := prototype.TimePointSec{UtcSeconds: t}
	return &val
}
