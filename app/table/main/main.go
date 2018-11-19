package main

import (
	"fmt"
	"github.com/coschain/contentos-go/app/table"
	"github.com/coschain/contentos-go/iservices"
	"github.com/coschain/contentos-go/prototype"
	"github.com/coschain/contentos-go/db/storage"
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
	db,err := storage.NewDatabase("./demos/pbTool.db")
	if err != nil {
		return
	}
	db.Start(nil)
	//defer db.Close()

	//1.create the table wrap
	//we can use the type  which is contained in another created pb struct,
	// such as "prototype.account_name" in AccountName 、prototype.time_point_sec
	//MakeXXX func can create a pb struct
	mKey := prototype.MakeAccountName("myName")
	wrap := table.NewSoDemoWrap(db, mKey)
	if wrap == nil {
		//crreate fail , the db already contain table with current mainKey
		log.Println("crreate fail , the db already contain table with current mainKey")
		return
	}

    if wrap.CheckExist() == nil {
    	wrap.RemoveDemo()
	}
	 //2.save table data to db
	 err  = wrap.CreateDemo(func(tInfo *table.SoDemo) {
		 tInfo.Owner = mKey
		 tInfo.Title = "hello"
		 tInfo.Content = "test the pb tool"
		 tInfo.Idx = 1001
		 tInfo.LikeCount = 100
		 tInfo.Taglist = "#NBA"
		 tInfo.ReplayCount = 100
		 tInfo.PostTime = creTimeSecondPoint(20120401)
	 })
	 if err != nil {
		 fmt.Println("create new table of Demo fail")
		 return
	 }

	 /*
	   --------------------------
	   Get Property（the GetXXX function  return the property value）
	   --------------------------*/

	 //get title
	 var t string
	 err = wrap.GetTitle(&t)
	 if err == nil {
		 fmt.Printf("the title is %s \n",t)
	 }else {
		 fmt.Println("get title fail")
	 }

	 //get content
	 var c string
	 err = wrap.GetContent(&c)
	 if err == nil {
		fmt.Printf("the content is %s \n",c)
	}else {
		fmt.Println("modify tilte fail")
	}
	//modify title
	mErr := wrap.MdContent("hello world")
	if mErr != nil {
		fmt.Println("modify tilte fail")
	}


	/*
	  --------------------------
	   Modify property value (******can't modify the mainkey)
	  --------------------------*/
	//modify content
	mErr = wrap.MdContent("test md the content")
	if mErr != nil {
		fmt.Println("modify content fail")
	}


	/*--------------------------
	   Sort Query List
	  --------------------------*/
     //1.create the sort wrap for property which is surpport sort (E.g postTime)
	 tSortWrap := table.SDemoPostTimeWrap{}
	tSortWrap.Dba = db
	 //2.start query data of range(sort by order)
	//start = nil  end = nil (query the db from start to end)
	//start = nil (query from start the db)
	//end = nil (query to the end of db)
	iter := new(iservices.IDatabaseIterator)
	err = tSortWrap.QueryListByOrder(creTimeSecondPoint(20120401),
		creTimeSecondPoint(20120415),iter)
	//we can get the main key and sub key by the returned iterator
	//if query by order the start value can't greater than end value
	if iter != nil {
		for (*iter).Next() {
			//get the mainkey value (GetMainVal return the ptr of value)
			mKeyPtr := new(prototype.AccountName)
			err := tSortWrap.GetMainVal(*iter,&mKeyPtr)
			if err != nil {
				fmt.Println("get main key fail")
			}else {
				fmt.Printf("the main key is %s in range \n",mKeyPtr.Value)
			}
			//get subKey value (the postTime value)
			mSubPtr := new(prototype.TimePointSec)
			err = tSortWrap.GetSubVal(*iter,&mSubPtr)
			if err != nil {
				fmt.Println("get postTime fail")
			}else {
				fmt.Printf("the postTime is %d \n",mSubPtr.UtcSeconds)
			}
		}
		//******* we must delete the iterator after end of use,otherwise maybe cause unKnow error *******//
		tSortWrap.DelIterater(*iter)
	}else {
		fmt.Println("there is no data exist in range posttime")
	}

	//query by reverse order
	//start = nil  end = nil (query the db from start to end)
	//start = nil (query from start the db)
	//end = nil (query to the end of db)
	iter1 := new(iservices.IDatabaseIterator)
	err = tSortWrap.QueryListByRevOrder(creTimeSecondPoint(20120415),
		creTimeSecondPoint(20120401),iter1)
	//we can get the main key and sub key by the returned iterator
	//if query by reverse order the start value can't less than end value
	if err == nil && iter1 != nil {
		for (*iter1).Next() {
			mKeyPtr := new(prototype.AccountName)
			err := tSortWrap.GetMainVal(*iter,&mKeyPtr)
			tSortWrap.GetMainVal(*iter1,&mKeyPtr)
			if err != nil {
				fmt.Println("query by reverse order get main key fail")
			}else {
				fmt.Printf("the main key is %s in reverse order  \n",mKeyPtr.Value)
			}
			mSubPtr := new(prototype.TimePointSec)
			err = tSortWrap.GetSubVal(*iter1,&mSubPtr)
			if err != nil || mKeyPtr == nil {
				fmt.Println("query by reverse order get postTime fail")
			}else {
				fmt.Printf("the postTime is %d in reverse order \n",mSubPtr.UtcSeconds)
			}
			mKeyPtr = nil
		}
     //******** delete the iterator ***********//
		tSortWrap.DelIterater(*iter1)
	}else {
		fmt.Println("there is no data exist in reverse order")
	}


    //query single value but not a range,start and end set the same value
	iter2 := new(iservices.IDatabaseIterator)
	err = tSortWrap.QueryListByOrder(creTimeSecondPoint(20136666),
		creTimeSecondPoint(20136666),iter2)
	if err == nil && iter2 != nil {
		if (*iter2).Next() {
			mKeyPtr := new(prototype.AccountName)
			err := tSortWrap.GetMainVal(*iter2,&mKeyPtr)
			if err != nil || mKeyPtr == nil {
				fmt.Println("get main key fail in range")
			}
			mKeyPtr = nil
		}

		tSortWrap.DelIterater(*iter2)
	}

	//query without start
	iter3 := new(iservices.IDatabaseIterator)
	err = tSortWrap.QueryListByOrder(nil,creTimeSecondPoint(20120422),iter3)
	if err == nil && iter3 != nil  {
		for (*iter3).Next() {
			mKeyPtr := new(prototype.AccountName)
			err := tSortWrap.GetMainVal(*iter3,&mKeyPtr)
			if err != nil || mKeyPtr == nil{
				fmt.Println("get main key fail in range when query without start 1111")
			}else {
				fmt.Printf("the main key is %s in range when query without start  \n",mKeyPtr.Value)
			}
			mKeyPtr = nil
		}
		tSortWrap.DelIterater(*iter3)
	}else {
		fmt.Println("there is no data exist without start")
	}

	//query without end
	iter4 := new(iservices.IDatabaseIterator)
	err = tSortWrap.QueryListByOrder(nil,nil,iter4)
	if err == nil &&  iter4 != nil  {
		for (*iter4).Next() {
			mKeyPtr := new(prototype.AccountName)
			err := tSortWrap.GetMainVal(*iter4,&mKeyPtr)
			if err != nil || mKeyPtr == nil{
				fmt.Println("get main key fail in range when query without end")
			}else {
				fmt.Printf("the main key is %s in range when query without end \n",mKeyPtr.Value)
			}
			tSortWrap.DelIterater(*iter4)
			mKeyPtr = nil
		}

	}else {
		fmt.Println("there is no data in range when query without end")
	}

	//query without start and end
	iter5 := new(iservices.IDatabaseIterator)
	err = tSortWrap.QueryListByOrder(nil,nil,iter5)
	if err == nil && iter5 != nil {
		for (*iter5).Next() {
			mKeyPtr := new(prototype.AccountName)
			err := tSortWrap.GetMainVal(*iter5,&mKeyPtr)
			if err != nil || mKeyPtr == nil {
				fmt.Printf("get main key fail in range when query without start and end \n")
			} else {
				fmt.Printf("the main key is %s when query without start and end  \n", mKeyPtr.Value)
			}
			mKeyPtr = nil
		}
		tSortWrap.DelIterater(*iter5)
	}

	//query without start and end by reverse order
	iter6 := new(iservices.IDatabaseIterator)
	err = tSortWrap.QueryListByRevOrder(nil,nil,iter6)
	if err == nil && iter6 != nil {
		for (*iter6).Next() {
			mKeyPtr := new(prototype.AccountName)
			err := tSortWrap.GetMainVal(*iter6,&mKeyPtr)
			if err != nil || mKeyPtr == nil{
				fmt.Println("get main key fail in range when query without start and end by reverse sort ")
			}else {
				fmt.Printf("the main key is %s in range when query without start and end by reverse sort \n",mKeyPtr.Value)
			}
			mKeyPtr = nil
		}
		tSortWrap.DelIterater(*iter6)
	}else {
		fmt.Println("there is no data in reverse order without start and end")
	}

	/*
	 --------------------------
	  unique Query List (only support query the property which is flag unique)
	 --------------------------*/
	 //1.create the uni wrap of property which is need unique query
	 var idx int64 = 1001
	 //create the UniXXXWrap
	 uniWrap := table.UniDemoIdxWrap{}
	 //set the dataBase to UniXXXWrap
	 uniWrap.Dba = db
	 //2.use UniQueryXX func to query data meanWhile return the table wrap
	 dWrap := new(table.SoDemoWrap)
	 err = uniWrap.UniQueryIdx(&idx,dWrap)
	 if err != nil {
	 	fmt.Printf("uni query fail ,error:%s\n",err)
	 }else {
	 	 var title string
		 err := dWrap.GetTitle(&title)
		 if err == nil {
			 fmt.Printf("the title of index is %s \n",title)
		 }
	 }

	//unique query mainkey(E.g query owner)
	doWrap := table.SoDemoWrap{}
	mUniWrap := table.UniDemoOwnerWrap{}
	mUniWrap.Dba = db
	err = mUniWrap.UniQueryOwner(prototype.MakeAccountName("myName"),&doWrap)
	if err == nil {
		var idx int64
		err :=  doWrap.GetIdx(&idx)
		if err == nil {
			fmt.Printf("owner is test,the idx is %d \n",idx)
		}else {
			fmt.Println("get the idx fail")
		}

	}

	  /*
	    remove tabale data from db
	  */
	  //judge the table of current mainKey if is exist
	  err  = wrap.CheckExist()
	  if err == nil {
	  	 err := wrap.RemoveDemo()
	  	 if err != nil {
	  	 	fmt.Println("remove the table data fail")
		 }
	  }

	 db.Close()
}

func creTimeSecondPoint(t uint32) *prototype.TimePointSec {
	val := prototype.TimePointSec{UtcSeconds:t}
	return &val
}
